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

import sys

import utils


def main():
  # detect distros to instantiate the correct class
  if sys.version_info >= (3, 6):
    import distro
  else:
    import platform as distro

  distribution = distro.linux_distribution()
  distro_name = distribution[0].lower()
  distro_version = distribution[1].split('.')[0]
  DistroClass = None

  if 'red hat enterprise linux' in distro_name and distro_version == '6':
    from redhat import RedHat6Tests as DistroClass
  elif 'red hat enterprise linux' in distro_name:
    from redhat import RedHat7Tests as DistroClass
  elif 'centos' in distro_name and distro_version == '6':
    from centos import CentOS6Tests as DistroClass
  elif 'centos' in distro_name:
    from centos import CentOS7Tests as DistroClass
  elif 'debian' in distro_name:
    from debian import DebianTests as DistroClass
  elif 'ubuntu' in distro_name:
    from ubuntu import UbuntuTests as DistroClass
  elif 'suse' in distro_name:
    from suse import SuseTests as DistroClass
  elif 'FreeBSD' in distro.system():
    from freebsd import FreeBSDTests as DistroClass
  else:
    raise Exception('Distribution %s is not supported' % distro_name)

  instance_name = utils.MetadataManager.FetchMetadataDefault('instance_name')

  distro_tests = DistroClass()
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
