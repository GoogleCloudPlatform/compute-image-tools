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

import logging

import utils
import utils.diskutils as diskutils


class _Package:
  """A Zypper package to be installed.

  Attributes:
      name (str): Zypper's name for the package.
      gce (bool): Is the package specific to GCP? When set to true, this
                  package will only be installed if when `install_gce_packages`
                  is also true.
      required (bool): Can the workflow proceed if there's a failure to install
                       this package? When set to true, the workflow will
                       terminate if there's an error installing the package.
                       When set to false, the error is logged but the workflow
                       will continue.
  """

  def __init__(self, name, gce, required):
    self.name = name
    self.gce = gce
    self.required = required


class _SuseRelease:
  """Describes which packages and subscriptions are required for
     a particular SUSE release.

  Attributes:
      flavor (str): The SUSE release variant. Either `opensuse` or `sles`.
      major (str): The major release number. For example, for SLES 12 SP1,
                   the major version is 12.
      minor (str): The minor release number. For example, for SLES 12 SP1,
                   the major version is 1. SLES 12, it is 0.
      subscriptions (list of str): The subscriptions to be added.
  """

  def __init__(self, flavor, major, minor, subscriptions=None):
    self.flavor = flavor
    self.major = major
    self.minor = minor
    self.subscriptions = subscriptions


_packages = [
    _Package('cloud-init', gce=False, required=False),
    _Package('google-cloud-sdk', gce=True, required=False),
    _Package('google-compute-engine-init', gce=True, required=True),
    _Package('google-compute-engine-oslogin', gce=True, required=True)
]

_distros = [
    _SuseRelease(
        flavor='opensuse',
        major='15',
        minor='1',
    ),
    _SuseRelease(
        flavor='sles',
        major='15',
        minor='1',
        subscriptions=['sle-module-public-cloud/15.1/x86_64']
    ),
    _SuseRelease(
        flavor='sles',
        major='12',
        minor='5',
        subscriptions=['sle-module-public-cloud/12/x86_64']
    ),
]


def _get_distro(g) -> _SuseRelease:
  """Gets the SuseRelease object for the OS installed on the disk.

  Raises:
    ValueError: If there's not a SuseObject for the the OS on the disk.
  """
  for d in _distros:
    if d.flavor == g.gcp_image_distro:
      if d.major == g.gcp_image_major or d.major == '*':
        if d.minor == g.gcp_image_minor or d.minor == '*':
          return d
  raise ValueError('Import script not defined for {} {}.{}'.format(
      g.gcp_image_distro, g.gcp_image_major, g.gcp_image_minor))


def _install_subscriptions(distro, g):
  """Executes SuseConnect -p for each subscription on `distro`.

  Raises:
    ValueError: If there was a failure adding the subscription.
  """
  if distro.subscriptions:
    for subscription in distro.subscriptions:
      try:
        g.command(['SUSEConnect', '-p', subscription])
      except Exception as e:
        raise ValueError(
            'Command failed: SUSEConnect -p {}: {}'.format(subscription, e))


def _install_packages(distro, g, install_gce):
  """Installs the packages listed on `distro`.

  Respects the user's request of whether to include GCE packages
  via the `install_gce` argument.

  Raises:
    ValueError: If there's a failure to refresh zypper, or if there's
                a failure to install a required package.
  """
  try:
    g.command(['zypper', 'refresh'])
  except Exception as e:
    raise ValueError('Failed to call zypper refresh', e)
  for pkg in _packages:
    if pkg.gce and not install_gce:
      continue
    try:
      g.command(('zypper', '-n', 'install', '--no-recommends', pkg.name))
    except Exception as e:
      if pkg.required:
        raise ValueError(
            'Failed to install required package {}: {}'.format(pkg.name, e))
      else:
        logging.warning(
            'Failed to install optional package {}: {}'.format(pkg.name, e))


def _update_grub(g):
  """Update and rebuild grub to ensure image is bootable."""
  g.command(
      ['sed', '-i',
       r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
       '/etc/default/grub'])
  g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


def _reset_network(g):
  """Update network to use DHCP."""
  logging.info('Updating network to use DHCP.')
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
  distro = _get_distro(g)

  _install_virtio_drivers(g)
  _install_subscriptions(distro, g)
  _install_packages(distro, g, include_gce_packages)
  if include_gce_packages:
    logging.info('Enabling google services.')
    g.sh('systemctl enable /usr/lib/systemd/system/google-*')

  _reset_network(g)
  _update_grub(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(translate)
