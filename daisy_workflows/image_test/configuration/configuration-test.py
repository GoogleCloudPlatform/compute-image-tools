#!/usr/bin/env python2
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
import sys

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

  def EnsurePackageIsInstalled(self, package_name):
    """
    Raises exception if @package_name is not installed
    """
    if not self.IsPackageInstalled(package_name):
      raise Exception('%s package is not installed' % package_name)

  def TestNoIrqbalanceInstalled(self):
    """
    Ensure that `irqbalance` is not installed or running.
    """
    if self.IsPackageInstalled('irqbalance'):
      raise Exception('irqbalance should not be found')

  def TestConsoleLogging(self):
    """
    Ensure boot loader configuration for console logging is correct.
    """
    # TODO: Or maybe it can be removed as this is implicitly verified by other
    # tests, like os-login.
    pass

  def TestKernelCmdargs(self):
    """
    Ensure boot loader kernel command line args (per distro).
    """
    # TODO: more information needed. Follow up on:
    # https://github.com/GoogleCloudPlatform/compute-image-tools/issues/402
    pass

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
    Ensure that rsyslog is installed and configured (if the distro uses
    rsyslog) and that the hostname is properly set in the logs on boot.
    """
    if not self.IsPackageInstalled('rsyslog'):
      # rsyslog was not found, skip this test for this distro
      return

    # test if kernel and daemon messages are being logged to console. The
    # hostname output will be checked by the step "rsyslog-hostname-test"
    info = [
        ['kern.info', 'RsyslogKernelConsoleTest'],
        ['daemon.info', 'RsyslogDaemonConsoleTest'],
    ]
    for facility in info:
      utils.Execute(['logger', '-p'] + facility)

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
    utils.Execute(['grep', '^root:[\!*]', '/etc/shadow'])

  def TestSshdConfig(self):
    """
    Ensure sshd config has sane default settings
    """
    def parseSshdConfig(path):
      configs = {}
      with open(path) as f:
        for line in filter(RemoveCommentAndStrip, f.read().split('\n')):
          if line:
            # use line separator for key and # values
            entry = line.split(' ')
            # strip dictionary value
            configs[entry[0]] = ' '.join(entry[1:]).strip()
      return configs

    sshd_desired_configs = {
        'PermitRootLogin': 'no',
        'PasswordAuthentication': 'no',
    }

    actual_sshd_configs = parseSshdConfig('/etc/ssh/sshd_config')
    for key in sshd_desired_configs:
      if actual_sshd_configs[key] != sshd_desired_configs[key]:
        raise Exception('Sshd key "%s" should be "%s" and not "%s"' % (
            key, sshd_desired_configs[key], actual_sshd_configs[key]))

  def TestPackageManagerConfig(self):
    """
    Ensure apt/yum repos are setup for GCE repos.
    """
    # The google-cloud-sdk package should only be found if GCE repo is setup
    self.EnsurePackageIsInstalled('google-cloud-sdk')

  def TestNetworkInterfaceMTU(self):
    """
    Ensure that the network interface MTU is set to 1460.
    """
    cur_mtu = int(open('/sys/class/net/eth0/mtu').read())
    desired_mtu = 1460
    if cur_mtu != desired_mtu:
      raise Exception('Network MTU is %d instead of expected %d' % (
          cur_mtu, desired_mtu))

  def TestNTPConfig(self):
    """
    Ensure that the NTP server is set to metadata.google.internal.
    """
    # Below, not the most pythonic thing to do... but it's the easiest one
    command = ['grep', '^server\s\+metadata.google.internal', '/etc/ntp.conf']
    utils.Execute(command)
    # if the above command returned any lines, then it found a match

  @abc.abstractmethod
  def TestAutomaticSecurityUpdates(self):
    """
    Ensure automatic security updates are enabled per distro specs.
    """
    pass

  def TestSysctlSecurityParams(self):
    """
    Ensure sysctl security parameters are set.
    """
    def CheckSecurityParameter(key, desired_value):
      rc, output = utils.Execute(['sysctl', key], capture_output=True)
      actual_value = int(output.split(" = ")[1])
      if actual_value != desired_value:
        raise Exception('Security Parameter %s is %d but expected %d' % (
            key, actual_value, desired_value))

    CheckSecurityParameter('net.ipv4.ip_forward', 0)
    CheckSecurityParameter('net.ipv4.tcp_syncookies', 1)
    CheckSecurityParameter('net.ipv4.conf.all.accept_source_route', 0)
    CheckSecurityParameter('net.ipv4.conf.default.accept_source_route', 0)
    CheckSecurityParameter('net.ipv4.conf.all.accept_redirects', 0)
    CheckSecurityParameter('net.ipv4.conf.default.accept_redirects', 0)
    CheckSecurityParameter('net.ipv4.conf.all.secure_redirects', 1)
    CheckSecurityParameter('net.ipv4.conf.default.secure_redirects', 1)
    CheckSecurityParameter('net.ipv4.conf.all.send_redirects', 0)
    CheckSecurityParameter('net.ipv4.conf.default.send_redirects', 0)
    CheckSecurityParameter('net.ipv4.conf.all.rp_filter', 1)
    CheckSecurityParameter('net.ipv4.conf.default.rp_filter', 1)
    CheckSecurityParameter('net.ipv4.icmp_echo_ignore_broadcasts', 1)
    CheckSecurityParameter('net.ipv4.icmp_ignore_bogus_error_responses', 1)
    CheckSecurityParameter('net.ipv4.conf.all.log_martians', 1)
    CheckSecurityParameter('net.ipv4.conf.default.log_martians', 1)
    CheckSecurityParameter('net.ipv4.tcp_rfc1337', 1)
    CheckSecurityParameter('kernel.randomize_va_space', 2)

  def TestGcloudUpToDate(self):
    """
    Test for gcloud/gsutil (some distros won't have this) and validate that
    versions are up to date.

    https://github.com/GoogleCloudPlatform/compute-image-tools/issues/400
    """
    # firstly check if gcloud and gsutil are available
    rc_gcloud, output = utils.Execute(['gcloud', 'info'], raise_errors=False)
    rc_gsutil, output = utils.Execute(['gsutil', 'version'],
                                      raise_errors=False)
    if rc_gcloud != 0 or rc_gsutil != 0:
      # if these commands are not available, skip this test
      return

    # now test if their API are still valid
    utils.Execute(['gcloud', 'compute', 'images', 'list'])
    utils.Execute(['gsutil', 'ls'])


class RedHatTests(GenericDistroTests):
  def TestPackageInstallation(self):
    # install something to test repository sanity
    utils.Execute(['yum', '-y', 'install', 'tree'])
    # in case it was already installed, ask for reinstall just to be sure
    utils.Execute(['yum', '-y', 'reinstall', 'tree'])

  def IsPackageInstalled(self, package_name):
    # the following command returns zero if package is installed
    command = ['yum', '--assumeno', 'install', package_name]
    rc, output = utils.Execute(command, raise_errors=False)
    return rc == 0

  def TestAutomaticSecurityUpdates(self):
    # the following command returns zero if package is installed
    utils.Execute(['yum', '--assumeno', 'install', 'yum-cron'])

    # systemctl returns zero if service exists and is running
    utils.Execute(['systemctl', 'status', 'yum-cron'])

    # check yum-cron configuration
    configs = {
        'download_updates': 'yes',
        'apply_updates': 'yes',
    }
    for key in configs:
      command = ['grep', key, '/etc/yum/yum-cron.conf']
      rc, output = utils.Execute(command, capture_output=True)
      # get clean text after '=' token
      cur_value = RemoveCommentAndStrip(output[output.find('=') + 1:])
      if configs[key] != cur_value:
        raise Exception('Yum-cron config "%s" is "%s" but expected "%s"' % (
            key, cur_value, configs[key]))


class CentOSTests(RedHatTests):
  pass


class DebianTests(GenericDistroTests):
  def TestPackageInstallation(self):
    utils.Execute(['apt-get', 'update'])
    utils.Execute(['apt-get', 'install', '--reinstall', '-y', 'tree'])

  def IsPackageInstalled(self, package_name):
    # the following command returns zero if package is installed
    rc, output = utils.Execute(['dpkg', '-s', package_name],
                               raise_errors=False)
    return rc == 0

  def TestAutomaticSecurityUpdates(self):
    self.EnsurePackageIsInstalled('unattended-upgrades')

    # service returns zero if service exists and is running
    utils.Execute(['service', 'unattended-upgrades', 'status'])

    # check unattended upgrade configuration
    command = ['unattended-upgrade', '-v']
    rc, output = utils.Execute(command, capture_output=True)
    for line in output.split('\n'):
      token = "Allowed origins are:"
      if line.find(token) >= 0:
        if len(line) > len(token):
          # There is some repository used for unattended upgrades
          return
    raise Exception('No origin repository used by unattended-upgrade')


class UbuntuTests(DebianTests):
  pass


class SuseTests(GenericDistroTests):
  pass


def main():
  # detect distros to instantiate the correct class
  if sys.version_info >= (3, 6):
    import distro
  else:
    import platform as distro

  distribution = distro.linux_distribution()
  distro_name = distribution[0].lower()
  distro_tests = None

  if 'red hat enterprise linux' in distro_name:
    distro_tests = RedHatTests()
  elif 'centos' in distro_name:
    distro_tests = CentOSTests()
  elif 'debian' in distro_name:
    distro_tests = DebianTests()
  elif 'ubuntu' in distro_name:
    distro_tests = UbuntuTests()
  elif 'suse' in distro_name:
    distro_tests = SuseTests()
  else:
    raise Exception('Distribution %s is not supported' % distro_name)

  instance_name = utils.MetadataManager.FetchMetadataDefault('instance_name')

  distro_tests.TestPackageInstallation()
  distro_tests.TestNoIrqbalanceInstalled()
  distro_tests.TestConsoleLogging()
  distro_tests.TestKernelCmdargs()
  distro_tests.TestHostname(instance_name)
  distro_tests.TestRsyslogConfig()
  distro_tests.TestRootPasswordDisabled()
  distro_tests.TestSshdConfig()
  distro_tests.TestPackageManagerConfig()
  distro_tests.TestNetworkInterfaceMTU()
  distro_tests.TestNTPConfig()
  distro_tests.TestAutomaticSecurityUpdates()
  distro_tests.TestSysctlSecurityParams()
  distro_tests.TestGcloudUpToDate()


if __name__ == '__main__':
  utils.RunTest(main)
