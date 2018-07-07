#!/usr/bin/env python
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

"""Translate the EL image on a GCE VM.

Parameters (retrieved from instance metadata):

debian_release: The version of the distro (stretch)
install_gce_packages: True if GCE agent and SDK should be installed
"""

import logging

import utils

utils.AptGetInstall(['python-guestfs'])

import guestfs


google_cloud = '''
deb http://packages.cloud.google.com/apt cloud-sdk-{deb_release} main
deb http://packages.cloud.google.com/apt google-compute-engine-{deb_release}-stable main
deb http://packages.cloud.google.com/apt google-cloud-packages-archive-keyring-{deb_release} main
'''

interfaces = '''
source-directory /etc/network/interfaces.d
auto lo
iface lo inet loopback
auto eth0
iface eth0 inet dhcp
'''

def DistroSpecific(g):
  install_gce = utils.GetMetadataParam('install_gce_packages')
  deb_release = utils.GetMetadataParam('debian_release')

  if install_gce == 'true':
    logging.info('Installing GCE packages.')
    g.command(
        ['wget', 'https://packages.cloud.google.com/apt/doc/apt-key.gpg',
        '-O', '/tmp/gce_key'])
    g.command(['apt-key', 'add', '/tmp/gce_key'])
    g.rm('/tmp/gce_key')
    g.write(
        '/etc/apt/sources.list.d/google-cloud.list',
        google_cloud.format(deb_release=deb_release))
    # Remove Azure agent.
    try:
      g.command(['apt-get', 'remove', '-y', '-f', 'waagent', 'walinuxagent'])
    except Exception as e:
      logging.debug(str(e))
      logging.warn('Could not uninstall Azure agent. Continuing anyway.')

    g.command(['apt-get', 'update'])
    g.sh(
        'DEBIAN_FRONTEND=noninteractive '
        'apt-get install --assume-yes --no-install-recommends '
        'google-cloud-packages-archive-keyring google-cloud-sdk '
        'google-compute-engine python-google-compute-engine '
        'python3-google-compute-engine')

  # Update grub config to log to console.
  g.command(
      ['sed', '-i',
      r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '/etc/default/grub'])

  # Disable predictive network interface naming in Stretch.
  if deb_release == 'stretch':
    g.command(
        ['sed', '-i',
        r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 net.ifnames=0 biosdevname=0"#',
        '/etc/default/grub'])

  g.command(['update-grub2'])

  # Reset network for DHCP.
  logging.info('Resetting network to DHCP for eth0.')
  g.write('/etc/network/interfaces', interfaces)


def main():
  g = utils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  utils.UnmountDisk(g)

if __name__=='__main__':
  utils.RunTranslate(main) 
