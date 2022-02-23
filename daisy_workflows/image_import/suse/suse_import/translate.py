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
import typing

import guestfs
from on_demand import migrate
import utils
from utils import configs, diskutils
from utils.guestfsprocess import run


class _Tarball:
  """ Metadata for downloading and verifying a tarball.

  Attributes:
      url: The HTTP or HTTPS URL for the tarball.
      sha256: The sha256 checksum for the tarball.
  """

  def __init__(self, url: str, sha256: str):
    self.url = url
    self.sha256 = sha256


class _SuseRelease:
  """Describes packages and products required for a particular SUSE release.

  Attributes:
      product: The SUSE release variant. One of: `opensuse` or `sles`.
      major: The major release number. For example, for SLES 12 SP1,
          the major version is 12.
      minor: The minor release number. For example, for SLES 12 SP1,
          the major version is 1. SLES 12, it is 0.
      cloud_product: The SCC product required to access cloud-related packages.
      on_demand_rpms: Tarball of RPMs to use when converting to
          on-demand licensing.
      gce_packages: List of packages required to install
          the GCE environment.
  """

  def __init__(self, product: str, major: str, minor: str = None,
               cloud_product: str = '',
               on_demand_rpms: _Tarball = None,
               gce_packages: typing.List[str] = None):
    self.name = product
    self.major = major
    self.minor = minor
    self.cloud_product = cloud_product
    self.on_demand_rpms = on_demand_rpms
    self.gce_packages = gce_packages

  def __repr__(self):
    if self.minor:
      return '{}-{}.{}'.format(self.name, self.major, self.minor)
    else:
      return '{}-{}'.format(self.name, self.major)


_on_demand_rpm_pattern = (
    'https://storage.googleapis.com/compute-image-tools/linux_import_tools/'
    'sles/{timestamp}/late_instance_offline_update_gce_SLE{major}.tar.gz')

# Packages for the GCE environment that are available in SLES's repo
# (OpenSUSE still has the older google-compute-engine-init environment).
#
# rsyslog is required by the guest environment to print to the serial console.
# It's not expressed as an RPM dependency, since the official images always
# include it.
_sles_gce_packages = [
  'google-guest-agent',
  'google-guest-configs',
  'google-guest-oslogin',
  'google-osconfig-agent',
  'rsyslog'
]

# Release versions supported by this importer. The flavor, major, and minor
# attributes are evaluated as regex patterns against the detected OS's name
# and major and minor version.
_releases = [
    _SuseRelease(
        product='opensuse',
        major='15',
        minor='1|2',
        gce_packages=[
          'google-compute-engine-init',
          'google-compute-engine-oslogin',
          'rsyslog'
        ]
    ),
    _SuseRelease(
        product='sles',
        major='15',
        minor='1|2|3',
        cloud_product='sle-module-public-cloud/{major}.{minor}/x86_64',
        on_demand_rpms=_Tarball(
            _on_demand_rpm_pattern.format(major=15, timestamp=1599058662),
            '156e43507d4c74c532b2f03286f47a2b609d32521be3e96dbc8fbad0b2976231'
        ),
        gce_packages=_sles_gce_packages
    ),
    _SuseRelease(
        product='sles',
        major='15',
        minor='0',
        cloud_product='sle-module-public-cloud/{major}/x86_64',
        on_demand_rpms=_Tarball(
            _on_demand_rpm_pattern.format(major=15, timestamp=1599058662),
            '156e43507d4c74c532b2f03286f47a2b609d32521be3e96dbc8fbad0b2976231'
        ),
        gce_packages=_sles_gce_packages
    ),
    _SuseRelease(
        product='sles',
        major='12',
        minor='4|5',
        cloud_product='sle-module-public-cloud/{major}/x86_64',
        on_demand_rpms=_Tarball(
            _on_demand_rpm_pattern.format(major=12, timestamp=1599058662),
            '9431afed6aaa7b79d68050c38cd8d2cbfebe9d03ea80b66c5f246cc99fb58491'
        ),
        gce_packages=_sles_gce_packages
    ),
]


def _get_release(g: guestfs.GuestFS) -> _SuseRelease:
  """Gets the _SuseRelease object for the OS installed on the disk.

  Raises:
    ValueError: If there's not a _SuseRelease for the the OS on the disk
    defined in _releases.
  """

  product = g.gcp_image_distro
  major = g.gcp_image_major
  minor = g.gcp_image_minor

  if 'unknown' == product:
    raise ValueError('No SUSE operating systems found.')

  matched = None
  for r in _releases:
    if re.match(r.name, product) \
        and re.match(r.major, major) \
        and re.match(r.minor, minor):
      matched = r
  if not matched:
    supported = ', '.join([d.__repr__() for d in _releases])
    raise ValueError(
        'Import of {}-{}.{} is not supported. '
        'The following versions are supported: [{}]'.format(
            product, major, minor, supported))
  return _SuseRelease(
      product=matched.name,
      major=major,
      minor=minor,
      cloud_product=matched.cloud_product.format(major=major, minor=minor),
      on_demand_rpms=matched.on_demand_rpms,
      gce_packages=matched.gce_packages
  )


def _disambiguate_suseconnect_product_error(
    g: guestfs.GuestFS, product: str, error: Exception) -> Exception:
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
def _install_product(g: guestfs.GuestFS, release: _SuseRelease):
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


@utils.RetryOnFailure(stop_after_seconds=5 * 60, initial_delay_seconds=1)
def _install_packages(g: guestfs.GuestFS, pkgs: typing.List[str]):
  if not pkgs:
    return
  try:
    g.sh('zypper --debug --non-interactive install --no-recommends '
         + ' '.join(pkgs))
  except Exception as e:
    raise ValueError('Failed to install {}: {}'.format(pkgs, e))


@utils.RetryOnFailure(stop_after_seconds=5 * 60, initial_delay_seconds=1)
def _refresh_zypper(g: guestfs.GuestFS):
  try:
    g.command(['zypper', '--debug', 'refresh'])
  except Exception as e:
    raise ValueError('Failed to call zypper refresh', e)


def _update_grub(g: guestfs.GuestFS):
  """Update and rebuild grub to ensure the image is bootable on GCP.
  See https://cloud.google.com/compute/docs/import/import-existing-image
  """
  g.write('/etc/default/grub', configs.update_grub_conf(
      g.cat('/etc/default/grub'),
      GRUB_CMDLINE_LINUX_DEFAULT='console=ttyS0,38400n8',
      GRUB_CMDLINE_LINUX='',
  ))
  g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


def _reset_network(g: guestfs.GuestFS):
  """Update network to use DHCP."""
  logging.info('Updating network to use DHCP.')
  if g.exists('/etc/resolv.conf'):
    g.sh('echo "" > /etc/resolv.conf')
  g.write('/etc/sysconfig/network/ifcfg-eth0', '\n'.join((
      'BOOTPROTO=dhcp',
      'STARTMODE=auto',
      'DHCLIENT_SET_HOSTNAME=yes')
  ))


def _install_virtio_drivers(g: guestfs.GuestFS):
  """Rebuilds initramfs to ensure that virtio drivers are present."""
  logging.info('Installing virtio drivers.')
  for kernel in g.ls('/lib/modules'):
    g.command(['dracut', '-v', '-f', '--kver', kernel])


def translate():
  """Mounts the disk, runs translation steps, then unmounts the disk."""
  include_gce_packages = utils.GetMetadataAttribute(
      'install_gce_packages', 'true').lower() == 'true'
  subscription_model = utils.GetMetadataAttribute(
      'subscription_model', 'byol').lower()

  g = diskutils.MountDisk('/dev/sdb')
  release = _get_release(g)

  pkgs = release.gce_packages if include_gce_packages else []

  if subscription_model == 'gce':
    logging.info('Converting to on-demand')
    migrate.migrate(
        g=g, tar_url=release.on_demand_rpms.url,
        tar_sha256=release.on_demand_rpms.sha256,
        cloud_product=release.cloud_product,
        post_convert_packages=pkgs
    )
  else:
    _install_product(g, release)
    _refresh_zypper(g)
    _install_packages(g, pkgs)
  _install_virtio_drivers(g)
  if include_gce_packages:
    logging.info('Enabling google services.')
    for unit in g.ls('/usr/lib/systemd/system/'):
      if unit.startswith('google-'):
        run(g, ['systemctl', 'enable', '/usr/lib/systemd/system/' + unit],
            raiseOnError=True)

  _reset_network(g)
  _update_grub(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


def main():
  utils.RunTranslate(translate, run_with_tracing=False)
