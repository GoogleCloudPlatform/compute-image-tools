#!/usr/bin/env python3
# Copyright 2018 Google Inc. All Rights Reserved.
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

import generic_distro
import utils


class FreeBSDTests(generic_distro.GenericDistroTests):
  def TestPackageInstallation(self):
    utils.Execute(['pkg', 'install', '--force', '--yes', 'tree'])

  def IsPackageInstalled(self, package_name):
    # the following command returns zero if package is installed
    command = ['pkg', 'query', '%n', package_name]
    rc, output = utils.Execute(command, raise_errors=False)
    return rc == 0

  def GetCmdlineConfigs(self):
    return {
        'console': ['vidconsole', ],
        # 'comconsole_speed': ['38400', ]
    }

  def GetCmdlineLocation(self):
    return '/boot/loader.conf'

  def GetSshdConfig(self):
    # Don't check for PermitRootLogin and PasswordAuthentication as it
    # fallbacks on FreeBSD to "no" when undefined
    return {}

  def TestRootPasswordDisabled(self):
    """
    Ensure root password is disabled (/etc/passwd)
    """
    # It's actually empty and it's fine according to:
    # https://forums.freebsd.org/threads/jails-default-root-password-is-empty-not-starred-out.37701/
    utils.Execute(['grep', '^root::', '/etc/master.passwd'])

  def TestPackageManagerConfig(self):
    """
    BSD has all GCE packages officially on its mirror
    """
    pass

  def TestNetworkInterfaceMTU(self):
    """
    Ensure that the network interface MTU is set to 1460.
    """
    # Parsing from ifconfig as BSD has no sysfs
    rc, output = utils.Execute(['ifconfig'], capture_output=True)
    for line in output.split('\n'):
      token = 'mtu '
      token_pos = line.find(token)
      if token_pos >= 0:
        desired_mtu = 1460
        cur_mtu = int(line[token_pos + len(token):])
        if cur_mtu != desired_mtu:
          raise Exception('Network MTU is %d but expected %d' % (
              cur_mtu, desired_mtu))

  def TestAutomaticSecurityUpdates(self):
    def HasFoundConfig(config_file, key, value):
      """
      Return True if @value is found inside of a @key line on @config_file.
      """
      command = ['grep', key, config_file]
      rc, output = utils.Execute(command, capture_output=True)
      output_lines = output.split('\n')
      useful_lines = filter(generic_distro.RemoveCommentAndStrip, output_lines)
      for line in useful_lines:
        if line.find(value) >= 0:
          # found desired value
          return True
      return False

    config_file = '/etc/freebsd-update.conf'
    desired_configs = {
        'Components': 'kernel',
    }

    for key in desired_configs:
      if not HasFoundConfig(config_file, key, desired_configs[key]):
        raise Exception('freebsd-update "%s" config has no "%s" on it' % (
            key, desired_configs[key]))

  def GetSysctlConfigs(self):
    """
    Return BSD parameters for sysctl checks.
    Below is what I could find on BSD related to original Linux's requirements.
    """
    return {
        'net.inet.ip.forwarding': 0,
        'net.inet.tcp.syncookies': 1,
        'net.inet.ip.accept_sourceroute': 0,
        'net.inet.ip.redirect': 0,
        'net.inet.icmp.bmcastecho': 0,
    }
