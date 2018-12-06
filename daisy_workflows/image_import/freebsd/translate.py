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

"""Translate the FreeBSD image on a GCE VM.

Parameters (retrieved from instance metadata):

install_gce_packages: True if GCE agent and SDK should be installed
"""

import glob
import logging
import os
import subprocess

import utils


rc_conf = '''
ifconfig_re0="DHCP"
sshd_enable="YES"
ifconfig_DEFAULT="SYNCDHCP mtu 1460"
ntpd_sync_on_start="YES"
'''

ntp_conf = '''
server metadata.google.internal iburst
'''

console_settings = '''
console="comconsole,vidconsole"
comconsole_speed="38400"
autoboot_delay="-1"
loader_logo="none"
'''

google_services = '''
google_startup_enable=YES
google_accounts_daemon_enable=YES
google_clock_skew_daemon_enable=YES
google_instance_setup_enable=YES
google_ip_forwarding_daemon_enable=YES
google_network_setup_enable=YES
'''


def DistroSpecific(c):
  install_gce = utils.GetMetadataAttribute('install_gce_packages')

  def UpdateConfigs(config, filename):
    """
    Parses @config and removes left side of '=' char from @filename and then
    add the full @config to @filename
    """
    # Removing, if it exists
    if os.path.isfile(filename):
      for t in config.split('\n'):
        if t:
          to_be_removed = t.split('=')[0]
          c.sh('sed -i -e "/%(pattern)s/d" %(filename)s' % {
              'pattern': to_be_removed,
              'filename': filename,
          })
    else:
      # Indicate that a new one is being created throught a comment
      with open('/etc/resolv.conf', 'w') as f:
        f.write("# Created by daisy's image import\n")

    # Adding
    c.write_append(config, filename)

  UpdateConfigs(rc_conf, '/etc/rc.conf')

  if install_gce == 'true':
    c.sh('ASSUME_ALWAYS_YES=yes pkg update')

    logging.info('Installing GCE packages.')
    c.sh('pkg install --yes py27-google-compute-engine google-cloud-sdk')

    # Activate google services
    UpdateConfigs(google_services, '/etc/rc.conf')

  # Update console configuration on boot
  UpdateConfigs(console_settings, '/boot/loader.conf')

  # Update device names on fstab
  c.sh('sed -i -e "s#/dev/ada#/dev/da#" /etc/fstab')

  # Remove any hard coded DNS settings in resolvconf.
  logging.info('Resetting resolvconf base.')
  with open('/etc/resolv.conf', 'w') as f:
    f.write('\n')


class Chroot(object):
  """
  Simple alternative to guestfs lib, as it doesn't exist in FreeBSD
  """
  def __init__(self, device):
    self.mount_point = '/translate'
    os.mkdir(self.mount_point)

    def FindAndMountRootPartition():
      """
      Try to mount partitions of @device onto @self.mount_point. The root
      partition should have a 'bin' folder. Return false if not found.
      """
      for part in glob.glob(device + 'p*'):
        try:
          self.sh('mount %s %s' % (part, self.mount_point))

          # check if a bin folder exists
          if os.path.isdir(os.path.join(self.mount_point, 'bin')):
            # Great! Found a desired partition with rootfs
            return True

          # No bin folder? Try the next partition
          self.sh('umount %s' % self.mount_point)

        except Exception as e:
          logging.info('Failed to mount %s. Reason: %s. Continuing...' % (
            part, e))

      # Too bad. Didn't find one
      return False

    if not FindAndMountRootPartition():
      raise Exception("No root partition found on disk %s" % device)

    # copy resolv.conf for using internet connection before chroot
    self.sh('cp /etc/resolv.conf %s/etc/resolv.conf' % self.mount_point)

    # chroot to that environment
    os.chroot(self.mount_point)

  def sh(self, cmd):
    p = subprocess.Popen(cmd, shell=True)
    p.communicate()
    returncode = p.returncode
    if returncode != 0:
      raise subprocess.CalledProcessError(returncode, cmd)

  def write_append(self, content, filename):
    with open(filename, 'a') as dst:
      dst.write(content)


def main():
  # Class will mount appropriate partition on da1 disk
  c = Chroot('/dev/da1')
  DistroSpecific(c)

  # Remove SSH host keys.
  logging.info('Removing SSH host keys.')
  c.sh("rm -f /etc/ssh/ssh_host_*")


if __name__ == '__main__':
  utils.RunTranslate(main)
