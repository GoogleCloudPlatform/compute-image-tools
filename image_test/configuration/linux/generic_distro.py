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

import abc
import time

import utils


def RemoveCommentAndStrip(string):
  token = string.find('#')
  return string[:token].strip() if token >= 0 else string.strip()


class GenericDistroTests(object):
  """
  Tests that uses common linux environment commands and are not specific to any
  distribution in particular.

  The abstract methods were defined to force distribution-specific tests.
  """
  __metaclass__ = abc.ABCMeta

  @abc.abstractmethod
  def TestPackageInstallation(self):
    """
    Ensure a package can be installed from distro archives (`make` or any other
    generic package).
    """
    pass

  @abc.abstractmethod
  def IsPackageInstalled(self, package_name):
    """
    Returns True if @package_name is installed on system, otherwise False.
    """
    pass

  def TestNoIrqbalanceInstalled(self):
    """
    Ensure that `irqbalance` is not installed or running.
    """
    if self.IsPackageInstalled('irqbalance'):
      raise Exception('irqbalance should not be found')

  def GetCmdlineConfigs(self):
    """
    Return command line configurations to be checked.
    """
    return {
        'console': ['ttyS0', '38400n8'],
    }

  def GetCmdlineLocation(self):
    """
    Return the path for kernel arguments given by the bootloader
    """
    return '/proc/cmdline'

  def TestKernelCmdargs(self):
    """
    Ensure boot loader configuration for console logging is correct.
    Ensure boot loader kernel command line args (per distro).
    """
    def ReadCmdline():
      cmdline = open(self.GetCmdlineLocation()).read()
      # Store values as: { # e.g: "console=ttyS0,38400n8 ro"
      #   'console': ['ttyS0', '38400n8'],
      #   'ro': [],
      # }
      configs = {}
      args = []
      for line in cmdline.split('\n'):
        if line:
          args.extend(line.split(' '))
      for arg in args:
        v = arg.split('=')
        if len(v) > 1:
          configs[v[0]] = [i.replace('"', '').strip() for i in v[1].split(',')]
        else:
          configs[v[0]] = []
      return configs

    desired_configs = self.GetCmdlineConfigs()
    cur_configs = ReadCmdline()

    try:
      for desired_config, desired_values in desired_configs.items():
        for desired_value in desired_values:
          cur_value = cur_configs[desired_config]
          if desired_value:
            if desired_value not in cur_value:
              e = 'Desired cmdline arg "%s" with value "%s" not found in "%s"'
              raise Exception(e % (desired_config, desired_value, cur_value))
          else:
            # empty list
            if cur_value:
              e = 'Desired cmdline arg "%s" should not be defined as "%s"'
              raise Exception(e % (desired_config, cur_value))
    except KeyError as e:
      raise Exception('Desired cmdline arg "%s" not found' % e.args[0])

  def TestHostname(self, expected_hostname):
    """
    Ensure hostname gets set to the instance name.
    """
    import socket
    actual_hostname = socket.gethostname()
    if expected_hostname != actual_hostname:
      raise Exception('Hostname "%s" differs from expected "%s"' % (
          actual_hostname, expected_hostname))

  def TestRsyslogConfig(self):
    """
    Ensure that rsyslog is installed and configured and that the hostname is
    properly set in the logs on boot.
    """
    # test if kernel and daemon messages are being logged to console. The
    # hostname output will be checked by the step "rsyslog-hostname-test"
    info = [
        ['kern.info', 'RsyslogKernelConsoleTest'],
        ['daemon.info', 'RsyslogDaemonConsoleTest'],
    ]
    # Avoid log output overload on centos-6
    time.sleep(0.1)
    for facility in info:
      utils.Execute(['logger', '-p'] + facility)
      time.sleep(0.1)

  def TestRootPasswordDisabled(self):
    """
    Ensure root password is disabled (/etc/passwd)
    """
    # as 'man shadow' described:
    # If the password field contains some string that is not a valid result of
    # crypt(3), for instance ! or *, the user will not be able to use a unix
    # password to log in
    #
    # Below, not the most pythonic thing to do... but it's the easiest one
    utils.Execute(['grep', '^root:[\\!*]', '/etc/shadow'])

  def GetSshdConfig(self):
    """
    Return desired sshd config to be checked.
    """
    return {
        'PermitRootLogin': 'no',
        'PasswordAuthentication': 'no',
    }

  def TestSshdConfig(self):
    """
    Ensure sshd config has sane default settings
    """
    def ParseSshdConfig(path):
      configs = {}
      with open(path) as f:
        # Avoid log output overload on centos-6
        time.sleep(0.1)
        for line in filter(RemoveCommentAndStrip, f.read().split('\n')):
          if line:
            # use line separator for key and # values
            entry = line.split(' ')
            # strip dictionary value
            configs[entry[0]] = ' '.join(entry[1:]).strip()
      return configs

    actual_sshd_configs = ParseSshdConfig('/etc/ssh/sshd_config')
    for desired_key, desired_value in self.GetSshdConfig().items():
      if actual_sshd_configs[desired_key] != desired_value:
        raise Exception('Sshd key "%s" should be "%s" and not "%s"' % (
            desired_key, desired_value, actual_sshd_configs[desired_key]))

  @abc.abstractmethod
  def TestPackageManagerConfig(self):
    """
    Ensure apt/yum repos are setup for GCE repos.
    """
    pass

  def TestNetworkInterfaceMTU(self):
    """
    Ensure that the network interface MTU is set to 1460.
    """
    from os import listdir
    for interface in listdir('/sys/class/net/'):
      if interface == 'lo':
        # Loopback is not subject to this restriction
        continue

      cur_mtu = int(open('/sys/class/net/%s/mtu' % interface).read())
      desired_mtu = 1460
      if cur_mtu != desired_mtu:
        raise Exception('Network MTU is %d but expected %d on %s interface' % (
            cur_mtu, desired_mtu, interface))

  def TestNTPConfig(self):
    """
    Ensure that the NTP server is set to metadata.google.internal.
    """
    def CheckNtpRun(cmd):
      """
      Run @cmd and check, if successful, whether google server is found on
      output.

      Args:
        cmd: list of strings. Command to be passed to utils.Exceute

      Return value:
        bool. True if client exists and google server is found. False
        otherwise.
      """
      try:
        rc, out = utils.Execute(cmd, raise_errors=False, capture_output=True)
        if rc == 0:
          # ntp client found on system
          if out.find('metadata.google') >= 0:
            # Google server found
            return True
      except OSError:
        # just consider it as a regular error as below
        pass

      # Command didn't run successfully
      return False

    # Try ntp
    if CheckNtpRun(['ntpq', '-p']):
      return

    # Try chrony
    if CheckNtpRun(['chronyc', 'sources']):
      return

    raise Exception("No NTP client found that uses Google's NTP server")

  @abc.abstractmethod
  def TestAutomaticSecurityUpdates(self):
    """
    Ensure automatic security updates are enabled per distro specs.
    """
    pass

  def GetSysctlConfigs(self):
    """
    Return linux parameters for sysctl checks.
    """
    return {
        'net.ipv4.ip_forward': 0,
        'net.ipv4.tcp_syncookies': 1,
        'net.ipv4.conf.all.accept_source_route': 0,
        'net.ipv4.conf.default.accept_source_route': 0,
        'net.ipv4.conf.all.accept_redirects': 0,
        'net.ipv4.conf.default.accept_redirects': 0,
        'net.ipv4.conf.all.secure_redirects': 1,
        'net.ipv4.conf.default.secure_redirects': 1,
        'net.ipv4.conf.all.send_redirects': 0,
        'net.ipv4.conf.default.send_redirects': 0,
        'net.ipv4.conf.all.rp_filter': 1,
        'net.ipv4.conf.default.rp_filter': 1,
        'net.ipv4.icmp_echo_ignore_broadcasts': 1,
        'net.ipv4.icmp_ignore_bogus_error_responses': 1,
        'net.ipv4.conf.all.log_martians': 1,
        'net.ipv4.conf.default.log_martians': 1,
        'net.ipv4.tcp_rfc1337': 1,
        'kernel.randomize_va_space': 2,
    }

  def TestSysctlSecurityParams(self):
    """
    Ensure sysctl security parameters are set.
    """
    def CheckSecurityParameter(key, desired_value):
      rc, output = utils.Execute(['sysctl', '-e', key], capture_output=True)
      actual_value = int(output.split("=")[1])
      if actual_value != desired_value:
        raise Exception('Security Parameter %s is %d but expected %d' % (
            key, actual_value, desired_value))

    sysctl_configs = self.GetSysctlConfigs()
    for config in sysctl_configs:
      # Avoid log output overload on centos-6
      time.sleep(0.1)
      CheckSecurityParameter(config, sysctl_configs[config])

  def TestGcloudUpToDate(self):
    """
    Test for gcloud/gsutil (some distros won't have this) and validate that
    versions are up to date.

    https://github.com/GoogleCloudPlatform/compute-image-tools/issues/400
    """
    # firstly check if gcloud and gsutil are available
    try:
      rc_gcloud, output = utils.Execute(['gcloud'], raise_errors=False)
      rc_gsutil, output = utils.Execute(['gsutil'], raise_errors=False)
    except OSError as e:
      if e.errno == 2:  # No such file or directory
        # command is not available, skip this test
        return
      raise e

    # Avoid log output overload on centos-6
    time.sleep(1)
    # now test if their API are still valid
    utils.Execute(['gcloud', 'compute', 'images', 'list'])

    # Avoid log output overload on centos-6
    time.sleep(1)
    utils.Execute(['gcloud', 'storage', 'ls'])
    time.sleep(1)
