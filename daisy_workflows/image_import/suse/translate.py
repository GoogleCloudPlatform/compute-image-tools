#!/usr/bin/env python3
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Translate the SUSE image on a GCE VM.

Parameters (retrieved from instance metadata):
  install_gce_packages: True if GCE agent and SDK should be installed
  licensing: Applicable for SLES. Either `gcp` or `byol`.
"""

import json
import logging
import re

import utils
import utils.configs as configs
import utils.diskutils as diskutils


class _Package:
  """A Zypper package to be installed.

  Attributes:
      name: Zypper's name for the package.
      gce: Is the package specific to GCP? When set to true, this
           package will only be installed if when `install_gce_packages`
           is also true.
  """

  def __init__(self, name: str, gce: bool):
    self.name = name
    self.gce = gce


class _SuseRelease:
  """Describes packages and products required for a particular SUSE release.

  Attributes:
      flavor: The SUSE release variant. Either `opensuse` or `sles`.
      major: The major release number. For example, for SLES 12 SP1,
          the major version is 12.
      minor: The minor release number. For example, for SLES 12 SP1,
          the major version is 1. SLES 12, it is 0.
      cloud_product: The SCC product required to access cloud-related packages.
  """

  def __init__(self, flavor: str, major: str, minor: str = None,
               cloud_product: str = ''):
    self.flavor = flavor
    self.major = major
    self.minor = minor
    self.cloud_product = cloud_product

  def __repr__(self):
    if self.minor:
      return '{}-{}.{}'.format(self.flavor, self.major, self.minor)
    else:
      return '{}-{}'.format(self.flavor, self.major)


_packages = [
    _Package('google-compute-engine-init', gce=True),
    _Package('google-compute-engine-oslogin', gce=True),
    # google-compute-engine-init configures rsyslog to show its
    # daemon logs (including the output of startup scripts)
    # to the serial console's output.
    _Package('rsyslog', gce=True)
]

_releases = [
    # Minor version omitted since libguestfs in Debian 9 doesn't recognize
    # opensuse 15.
    _SuseRelease(
        flavor='opensuse',
        major='15',
        minor='1|2',
    ),
    _SuseRelease(
        flavor='sles',
        major='15',
        minor='1|2',
        cloud_product='sle-module-public-cloud/{major}.{minor}/x86_64'
    ),
    _SuseRelease(
        flavor='sles',
        major='12',
        minor='4|5',
        cloud_product='sle-module-public-cloud/{major}/x86_64'
    ),
]


def _get_release(g) -> _SuseRelease:
  """Gets the _SuseRelease object for the OS installed on the disk.

  Raises:
    ValueError: If there's not a _SuseRelease for the the OS on the disk
    defined in _releases.
  """

  distro = g.gcp_image_distro
  major = g.gcp_image_major
  minor = g.gcp_image_minor

  matched = None
  for r in _releases:
    if re.match(r.flavor, distro) \
        and re.match(r.major, major) \
        and re.match(r.minor, minor):
      matched = r
  if not matched:
    supported = ', '.join([d.__repr__() for d in _releases])
    raise ValueError(
        'Import of {}-{}.{} is not supported. '
        'The following versions are supported: [{}]'.format(
            distro, major, minor,
            supported))
  return _SuseRelease(
      flavor=matched.flavor,
      major=major,
      minor=minor,
      cloud_product=matched.cloud_product.format(major=major, minor=minor)
  )


def _disambiguate_suseconnect_product_error(
    g, product: str, error: Exception) -> Exception:
  """Creates a user-debuggable error after failing to add a product
     using SUSEConnect.

  Args:
      g: Mounted GuestFS instance.
      product: The product that failed to be added.
      error: The error returned from `SUSEConnect -p`.
  """
  statuses = []
  try:
    statuses = json.loads(g.command(['SUSEConnect', '--status']))
  except Exception as e:
    return ValueError(
        'Unable to communicate with SCC. Ensure the import '
        'is running in a network that allows internet access.', e)

  # `SUSEConnect --status` returns a list of status objects,
  # where the triple of (identifier, version, arch) uniquely
  # identifies a product in SCC. Below are two examples.
  #
  # Example 1: SLES for SAP 12.2, No subscription
  # [
  #    {
  #       "identifier":"SLES_SAP",
  #       "version":"12.2",
  #       "arch":"x86_64",
  #       "status":"Not Registered"
  #    }
  # ]
  #
  # Example 2: SLES 15.1, Active
  # [
  #    {
  #       "status":"Registered",
  #       "version":"15.1",
  #       "arch":"x86_64",
  #       "identifier":"SLES",
  #       "subscription_status":"ACTIVE"
  #    },
  #    {
  #       "status":"Registered",
  #       "version":"15.1",
  #       "arch":"x86_64",
  #       "identifier":"sle-module-basesystem"
  #    }
  # ]

  for status in statuses:
    if status.get('identifier') not in ('SLES', 'SLES_SAP'):
      continue

    if status.get('subscription_status') == 'ACTIVE':
      return ValueError(
          'Unable to add product "%s" using SUSEConnect. Please ensure that '
          'your subscription includes access to this product.' % product,
          error)

  return ValueError(
      'Unable to find an active SLES subscription. '
      'SCC returned: %s' % statuses)


@utils.RetryOnFailure(stop_after_seconds=5 * 60, initial_delay_seconds=1)
def _install_product(g, release: _SuseRelease):
  """Executes SuseConnect -p for each product on `release`.

  Raises:
    ValueError: If there was a failure adding the subscription.
  """
  if release.cloud_product:
    try:
      g.command(['SUSEConnect', '--debug', '-p', release.cloud_product])
    except Exception as e:
      raise _disambiguate_suseconnect_product_error(
          g, release.cloud_product, e)


def _install_packages(g, install_gce):
  """Installs packages using zypper

  Respects the user's request of whether to include GCE packages
  via the `install_gce` argument.

  Raises:
    ValueError: If there's a failure to refresh zypper, or if there's
                a failure to install a required package.
  """
  refresh_zypper(g)
  to_install = []
  for pkg in _packages:
    if pkg.gce and not install_gce:
      continue
    to_install.append(pkg)
  install_packages(g, *to_install)


@utils.RetryOnFailure(stop_after_seconds=5 * 60, initial_delay_seconds=1)
def install_packages(g, *pkgs):
  try:
    g.sh('zypper --non-interactive install --no-recommends '
         + ' '.join([p.name for p in pkgs]))
  except Exception as e:
    raise ValueError('Failed to install {}: {}'.format(pkgs, e))


@utils.RetryOnFailure(stop_after_seconds=5 * 60, initial_delay_seconds=1)
def refresh_zypper(g):
  try:
    g.command(['zypper', 'refresh'])
  except Exception as e:
    raise ValueError('Failed to call zypper refresh', e)


def _update_grub(g):
  """Update and rebuild grub to ensure the image is bootable on GCP.
  See https://cloud.google.com/compute/docs/import/import-existing-image
  """
  g.write('/etc/default/grub', configs.update_grub_conf(
      g.cat('/etc/default/grub'),
      GRUB_CMDLINE_LINUX_DEFAULT='console=ttyS0,38400n8',
      GRUB_CMDLINE_LINUX='',
  ))
  g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


def _reset_network(g):
  """Update network to use DHCP."""
  logging.info('Updating network to use DHCP.')
  if g.exists('/etc/resolv.conf'):
    g.sh('echo "" > /etc/resolv.conf')
  g.write('/etc/sysconfig/network/ifcfg-eth0', '\n'.join((
      'BOOTPROTO=dhcp',
      'STARTMODE=auto',
      'DHCLIENT_SET_HOSTNAME=yes')
  ))


def _install_virtio_drivers(g):
  """Rebuilds initramfs to ensure that virtio drivers are present."""
  logging.info('Installing virtio drivers.')
  for kernel in g.ls('/lib/modules'):
    g.command(['dracut', '-v', '-f', '--kver', kernel])


def translate():
  """Mounts the disk, runs translation steps, then unmounts the disk."""
  include_gce_packages = utils.GetMetadataAttribute(
      'install_gce_packages', 'true').lower() == 'true'

  g = diskutils.MountDisk('/dev/sdb')
  release = _get_release(g)

  _install_product(g, release)
  _install_packages(g, include_gce_packages)
  _install_virtio_drivers(g)
  if include_gce_packages:
    logging.info('Enabling google services.')
    g.sh('systemctl enable /usr/lib/systemd/system/google-*')

  _reset_network(g)
  _update_grub(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(translate)
