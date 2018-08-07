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

"""Translate the SUSE image on a GCE VM.

Parameters (retrieved from instance metadata):

install_gce_packages: True if GCE agent and SDK should be installed
"""

import logging

import utils

utils.AptGetInstall(['python-guestfs'])


import guestfs


network = '''
BOOTPROTO='dhcp'
STARTMODE='auto'
DHCLIENT_SET_HOSTNAME='yes'
'''


def DistroSpecific(g):
  install_gce = utils.GetMetadataAttribute('install_gce_packages')

  # Remove any hard coded DNS settings in resolvconf.
  logging.info('Resetting resolvconf base.')
  g.sh('echo "" > /etc/resolv.conf')

  # Try to reset the network to DHCP.
  g.write('/etc/sysconfig/network/ifcfg-eth0', network)

  if install_gce == 'true':
    g.command(['zypper', 'refresh'])

    logging.info('Installing cloud-init.')
    g.command(['zypper', '-n', 'install', '--no-recommends', 'cloud-init'])

    # Installing google-compute-engine-init and not installing
    # gce-compute-image-packages as there is no port to SUSE
    logging.info('Installing GCE packages.')
    g.command(
        ['zypper', '-n', 'install', '--no-recommends',
        'google-compute-engine-init'])

    logging.info('Enable google services.')
    g.sh('systemctl enable /usr/lib/systemd/system/google-*')

    # Try to install the google-cloud-sdk package. It may be not available on
    # all Leap versions so don't raise an error if it fails.
    try:
      g.command(
          ['zypper', '-n', 'install', '--no-recommends',
          'google-cloud-sdk'])
    except Exception as e:
      print "Optional google-cloud-sdk package couldn't be installed: %s" % e

  # Update grub config to log to console, remove quiet and timeouts.
  g.command(
      ['sed', '-i',
      '-e', r's#^\(GRUB_CMDLINE_LINUX=".*\)"$#\1 console=ttyS0,38400n8"#',
      '-e', r's#^\(GRUB_CMDLINE_LINUX[^=]*=".*\)quiet\(.*"\)$#\1\2#',
      '-e', r's#^\(GRUB_[^=]*TIMEOUT\)=.*$#\1=0#',
      '/etc/default/grub'])

  g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])


def main():
  g = utils.MountDisk('/dev/sdb')
  DistroSpecific(g)
  utils.CommonRoutines(g)
  utils.UnmountDisk(g)


if __name__ == '__main__':
  utils.RunTranslate(main)
