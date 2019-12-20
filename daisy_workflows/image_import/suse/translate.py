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


class Package:
  name = ''
  gce = False
  required = False

  def __init__(self, name, gce, required):
    self.name = name
    self.gce = gce
    self.required = required


class SuseRelease:
  flavor = ''
  major = ''
  minor = ''
  packages = None
  subscriptions = None

  def __init__(self, flavor, major, minor, packages, subscriptions=None):
    self.flavor = flavor
    self.major = major
    self.minor = minor
    self.packages = packages
    self.subscriptions = subscriptions


cloud_init = Package('cloud-init', gce=False, required=False)
gcp_sdk = Package('google-cloud-sdk', gce=True, required=False)
gce_init = Package('google-compute-engine-init', gce=True, required=True)
gce_oslogin = Package('google-compute-engine-oslogin', gce=True, required=True)

_distros = [
    SuseRelease(
        flavor='opensuse',
        major='15',
        minor='1',
        packages=[cloud_init, gcp_sdk, gce_init, gce_oslogin]
    ),
    SuseRelease(
        flavor='sles',
        major='15',
        minor='1',
        packages=[cloud_init, gcp_sdk, gce_init, gce_oslogin],
        subscriptions=['sle-module-public-cloud/15.1/x86_64']
    ),
    SuseRelease(
        flavor='sles',
        major='12',
        minor='5',
        packages=[cloud_init, gcp_sdk, gce_init, gce_oslogin],
        subscriptions=['sle-module-public-cloud/12/x86_64']
    ),
]


def get_distro(g) -> SuseRelease:
  for d in _distros:
    if d.flavor == g.gcp_image_distro:
      if d.major == g.gcp_image_major or d.major == '*':
        if d.minor == g.gcp_image_minor or d.minor == '*':
          return d
  raise AssertionError('Import script not defined for {} {}.{}'.format(
      g.gcp_image_distro, g.gcp_image_major, g.gcp_image_minor))


def install_subscriptions(distro, g):
  if distro.subscriptions:
    for subscription in distro.subscriptions:
      try:
        g.command(['SUSEConnect', '-p', subscription])
      except Exception as e:
        raise ValueError(
            'Command failed: SUSEConnect -p {}: {}'.format(subscription, e))


def install_packages(distro, g, install_gce):
  g.command(['zypper', 'refresh'])
  for pkg in distro.packages:
    if pkg.gce and not install_gce:
      continue
    try:
      g.command(('zypper', '-n', 'install', '--no-recommends', pkg.name))
    except Exception as e:
      if pkg.required:
        raise AssertionError(
            'Failed to install required package {}: {}'.format(pkg.name, e))
      else:
        logging.warning(
            'Failed to install optional package {}: {}'.format(pkg.name, e))


def update_grub(g):
  g.command(
      ['sed', '-i',
       r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
       '/etc/default/grub'])
  g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


def reset_network(g):
  logging.info('Updating network to use DHCP.')
  g.sh('echo "" > /etc/resolv.conf')
  g.write('/etc/sysconfig/network/ifcfg-eth0', '\n'.join((
      'BOOTPROTO=dhcp',
      'STARTMODE=auto',
      'DHCLIENT_SET_HOSTNAME=yes')
  ))


def install_gcp_licensing(distro, g):
  if distro.flavor == 'opensuse':
    return
  licensing = utils.GetMetadataAttribute('licensing')
  if not licensing:
    raise AssertionError(
        'licensing attribute must be set to either gcp or byol.')
  if licensing == 'byol':
    return

  # TODO: Implement GCP licensing for SLES.
  # Here are SUSE's instructions:
  #   https://www.suse.com/c/create-sles-demand-images-gce/

  raise AssertionError(
      'GCP licensing not supported for imported SLES images. Please use BYOL.')


def install_virtio_drivers(g):
  logging.info('Installing virtio drivers.')
  for kernel in g.ls('/lib/modules'):
    g.command(['dracut', '-v', '-f', '--kver', kernel])


def translate():
  include_gce_packages = utils.GetMetadataAttribute(
      'install_gce_packages', 'true').lower() == 'true'

  g = diskutils.MountDisk('/dev/sdb')
  distro = get_distro(g)

  install_virtio_drivers(g)
  install_gcp_licensing(distro, g)
  install_subscriptions(distro, g)
  install_packages(distro, g, include_gce_packages)
  if include_gce_packages:
    logging.info('Enabling google services.')
    g.sh('systemctl enable /usr/lib/systemd/system/google-*')

  reset_network(g)
  update_grub(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(translate)
