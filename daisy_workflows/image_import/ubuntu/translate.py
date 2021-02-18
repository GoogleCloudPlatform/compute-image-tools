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

"""Translate the Ubuntu image on a GCE VM.

Parameters (retrieved from instance metadata):

ubuntu_release: The nickname of the distro (eg: trusty).
install_gce_packages: True if GCE agent and SDK should be installed
"""

from difflib import Differ
import logging
import re

import guestfs
import utils
import utils.diskutils as diskutils
from utils.guestfsprocess import run

# Google Cloud SDK
#
# The official images provide the Google Cloud SDK.
#
# Starting at 18, it's installed using snap. Since guestfs
# issues commands via a chroot, we don't have access to the
# snapd daemon. Therefore we schedule the SDK to be installed
# using cloud-init on the first boot.
#
# Prior to 18, the official images installed the cloud SDK
# using a partner apt repo.
cloud_init_cloud_sdk = '''
snap:
   commands:
      00: snap install google-cloud-sdk --classic
'''

# Ubuntu standard path, from https://wiki.ubuntu.com/PATH.
std_path = (
  '/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin')

# systemd directive to include /snap/bin on the path for a systemd service.
# This path was included in the Python guest agent's unit files, but
# was removed in the NGA's unit files.
snap_env_directive = '\n'.join([
  '[Service]',
  'Environment=PATH=' + std_path
])

apt_cloud_sdk = '''
# Enabled for Google Cloud SDK
deb http://archive.canonical.com/ubuntu {ubuntu_release} partner
'''

# Repo Mirrors
#
# This configures apt to prefer GCE's apt repos. It mirrors what's
# provided on the official images in
# /etc/cloud/cloud.cfg.d/91-gce.cfg and
# /etc/cloud/cloud.cfg.d/91-gce-system.cfg.
cloud_init_repos = '''
datasource_list: [ GCE ]
system_info:
   package_mirrors:
     - arches: [i386, amd64]
       failsafe:
         primary: http://archive.ubuntu.com/ubuntu
         security: http://security.ubuntu.com/ubuntu
       search:
         primary:
           - http://%(region)s.gce.archive.ubuntu.com/ubuntu/
           - http://%(availability_zone)s.gce.clouds.archive.ubuntu.com/ubuntu/
           - http://gce.clouds.archive.ubuntu.com/ubuntu/
         security: []
     - arches: [armhf, armel, default]
       failsafe:
         primary: http://ports.ubuntu.com/ubuntu-ports
         security: http://ports.ubuntu.com/ubuntu-ports
'''


# Network configs
#
# cloud-init will overwrite these after performing its
# network detection. They're required, however, so that
# cloud init can reach the metadata server to determine
# that it's running on GCE.
#
# https://cloudinit.readthedocs.io/en/latest/topics/boot.html
#
# netplan is default starting at 18.
# https://ubuntu.com/blog/ubuntu-bionic-netplan

network_trusty = '''
# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto eth0
iface eth0 inet dhcp

source /etc/network/interfaces.d/*.cfg
'''

network_xenial = '''
# The loopback network interface
auto lo
iface lo inet loopback

# The primary network interface
auto ens4
iface ens4 inet dhcp

source /etc/network/interfaces.d/*.cfg
'''

network_netplan = '''
network:
  version: 2
  renderer: networkd
  ethernets:
    ens4:
      dhcp4: true
'''


def install_cloud_sdk(g: guestfs.GuestFS, ubuntu_release: str) -> None:
  """ Installs Google Cloud SDK, supporting apt and snap.

  Args:
    g: A mounted GuestFS instance.
    ubuntu_release: The release nickname (eg: trusty).
  """
  try:
    run(g, 'gcloud --version')
    logging.info('Found gcloud. Skipping installation of Google Cloud SDK.')
    return
  except RuntimeError:
    logging.info('Did not find previous install of gcloud.')

  if g.gcp_image_major < '18':
    g.write('/etc/apt/sources.list.d/partner.list',
            apt_cloud_sdk.format(ubuntu_release=ubuntu_release))
    utils.update_apt(g)
    utils.install_apt_packages(g, 'google-cloud-sdk')
    logging.info('Installed Google Cloud SDK with apt.')
    return

  # Starting at 18.04, Canonical installs the sdk using snap.
  # Running `snap install` directly is not an option here since it
  # requires the snapd daemon to be running on the guest.
  g.write('/etc/cloud/cloud.cfg.d/91-google-cloud-sdk.cfg',
          cloud_init_cloud_sdk)
  logging.info(
    'Google Cloud SDK will be installed using snap with cloud-init.')

  # Include /snap/bin in the PATH for startup and shutdown scripts.
  # This was present in the old guest agent, but lost in the new guest
  # agent.
  for p in ['/lib/systemd/system/google-shutdown-scripts.service',
            '/lib/systemd/system/google-startup-scripts.service']:
    logging.debug('[%s] Checking whether /bin/snap is on PATH.', p)
    if not g.exists(p):
      logging.debug('[%s] Skipping: Unit not found.', p)
      continue
    original_unit = g.cat(p)
    # Check whether the PATH is already set; if so, skip patching to avoid
    # overwriting existing directive.
    match = re.search('Environment=[\'"]?PATH.*', original_unit,
                      flags=re.IGNORECASE)
    if match:
      logging.debug('[%s] Skipping: PATH already defined in unit file: %s.', p,
                    match.group())
      continue
    # Add Environment directive to unit file, and show diff in debug log.
    patched_unit = original_unit.replace('[Service]', snap_env_directive)
    g.write(p, patched_unit)
    diff = '\n'.join(Differ().compare(original_unit.splitlines(),
                                      patched_unit.splitlines()))
    logging.debug('[%s] PATH not defined. Added:\n%s', p, diff)


def DistroSpecific(g):
  ubuntu_release = utils.GetMetadataAttribute('ubuntu_release')
  install_gce = utils.GetMetadataAttribute('install_gce_packages')

  # If present, remove any hard coded DNS settings in resolvconf.
  # This is a common workaround to include permanent changes:
  # https://askubuntu.com/questions/157154
  if g.exists('/etc/resolvconf/resolv.conf.d/base'):
    logging.info('Resetting resolvconf base.')
    run(g, 'echo "" > /etc/resolvconf/resolv.conf.d/base')

  # Reset the network to DHCP.
  if ubuntu_release == 'trusty':
    g.write('/etc/network/interfaces', network_trusty)
  elif ubuntu_release == 'xenial':
    g.write('/etc/network/interfaces', network_xenial)
  elif g.is_dir('/etc/netplan'):
    run(g, 'rm -f /etc/netplan/*')
    g.write('/etc/netplan/config.yaml', network_netplan)
    run(g, 'netplan apply')

  if install_gce == 'true':
    utils.update_apt(g)
    logging.info('Installing cloud-init.')
    utils.install_apt_packages(g, 'cloud-init')
    # Ubuntu 14.04's version of cloud-init doesn't have `clean`.
    if g.gcp_image_major > '14':
      run(g, 'cloud-init clean')

    # Remove cloud-init configs that may conflict with GCE's.
    #
    # - subiquity disables automatic network configuration
    #     https://bugs.launchpad.net/ubuntu/+source/cloud-init/+bug/1871975
    for cfg in [
        'azure', 'curtin', 'waagent', 'walinuxagent', 'aws', 'amazon',
        'subiquity'
    ]:
      run(g, 'rm -f /etc/cloud/cloud.cfg.d/*%s*' % cfg)

    remove_azure_agents(g)

    g.write('/etc/cloud/cloud.cfg.d/91-gce-system.cfg', cloud_init_repos)

    utils.install_apt_packages(g, 'gce-compute-image-packages')
    install_cloud_sdk(g, ubuntu_release)

  # Update grub config to log to console.
  run(g, [
      'sed', '-i',
      r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '/etc/default/grub'
  ])
  run(g, ['update-grub2'])


def remove_azure_agents(g):
  try:
    run(g, ['apt-get', 'remove', '-y', '-f', 'walinuxagent'])
  except Exception as e:
    logging.debug(str(e))

  try:
    run(g, ['apt-get', 'remove', '-y', '-f', 'waagent'])
  except Exception as e:
    logging.debug(str(e))


def main():
  g = diskutils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(main, run_with_tracing=False)
