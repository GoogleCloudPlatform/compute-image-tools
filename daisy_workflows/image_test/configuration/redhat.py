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

import generic_distro
import utils


class RedHatTests(generic_distro.GenericDistroTests):
  """
  Abstract class. Please use a derived one.
  """
  __metaclass__ = abc.ABCMeta

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

  def TestPackageManagerConfig(self):
    command = ['grep', '-r', 'packages.cloud.google.com', '/etc/yum.repos.d/']
    utils.Execute(command)

  @abc.abstractmethod
  def GetYumCronConfig(self):
    """
    Return the location of yum-cron configuration on the system and a
    configuration dictionary to be checked on
    """
    pass

  def TestAutomaticSecurityUpdates(self):
    # the following command returns zero if package is installed
    utils.Execute(['yum', '--assumeno', 'install', 'yum-cron'])

    # service returns zero if service exists and is running
    utils.Execute(['service', 'yum-cron', 'status'])

    # check yum-cron configuration
    # Now this part is, unfortunately, different between RedHat 6 and 7
    yum_cron_file, configs = self.GetYumCronConfig()

    for key in configs:
      command = ['grep', key, yum_cron_file]
      rc, output = utils.Execute(command, capture_output=True)
      # get clean text after '=' token
      cur_value = generic_distro.RemoveCommentAndStrip(
          output[output.find('=') + 1:]
      )
      if configs[key] != cur_value:
        raise Exception('Yum-cron config "%s" is "%s" but expected "%s"' % (
            key, cur_value, configs[key]))


class RedHat6Tests(RedHatTests):
  def GetYumCronConfig(self):
    return (
        '/etc/sysconfig/yum-cron',
        {
            'CHECK_ONLY': 'no',
            'DOWNLOAD_ONLY': 'no',
        }
    )


class RedHat7Tests(RedHatTests):
  def GetYumCronConfig(self):
    return (
        '/etc/yum/yum-cron.conf',
        {
            'download_updates': 'yes',
            'apply_updates': 'yes',
        }
    )
