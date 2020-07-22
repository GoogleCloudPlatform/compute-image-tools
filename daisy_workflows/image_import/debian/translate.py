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

"""Translate the EL image on a GCE VM.

Parameters (retrieved from instance metadata):

debian_release: The version of the distro (stretch)
install_gce_packages: True if GCE agent and SDK should be installed
"""

import logging

import utils
import utils.diskutils as diskutils

google_cloud = '''
deb http://packages.cloud.google.com/apt cloud-sdk-{deb_release} main
deb http://packages.cloud.google.com/apt google-compute-engine-{deb_release}-stable main
deb http://packages.cloud.google.com/apt google-cloud-packages-archive-keyring-{deb_release} main
'''  # noqa: E501

interfaces = '''
source-directory /etc/network/interfaces.d
auto lo
iface lo inet loopback
auto eth0
iface eth0 inet dhcp
'''


def DistroSpecific(g):
  install_gce = utils.GetMetadataAttribute('install_gce_packages')
  deb_release = utils.GetMetadataAttribute('debian_release')

  if install_gce == 'true':
    logging.info('Installing GCE packages.')

    utils.update_apt(g)
    utils.install_apt_packages(g, 'gnupg')

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

    utils.update_apt(g)
    pkgs = ['google-cloud-packages-archive-keyring', 'google-compute-engine']
    # Debian 8 differences:
    #   1. No NGE
    #   2. No Cloud SDK, since it requires Python 3.5+
    if deb_release == 'jessie':
      pkgs += ['python-google-compute-engine',
               'python3-google-compute-engine']
    else:
      pkgs += ['google-cloud-sdk']
    utils.install_apt_packages(g, *pkgs)

  # Update grub config to log to console.
  g.command(
      ['sed', '-i""',
      r'/GRUB_CMDLINE_LINUX/s#"$# console=ttyS0,38400n8"#',
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
  g = diskutils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(main)
