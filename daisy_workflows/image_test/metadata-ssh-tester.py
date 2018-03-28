#!/usr/bin/python
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


import utils

utils.AptGetInstall(['python-pip'])
utils.Execute(['pip', 'install', '--upgrade', 'google-api-python-client'])

from googleapiclient import discovery
import oauth2client.client

MM = utils.MetadataManager
MD = None


def SetBlockProjectSshKeys(state):
  MD.Define('block-project-ssh-keys', state, MM.INSTANCE_LEVEL)


def TestLoginSshKeys(level):
  key = MD.AddSshKey(MM.SSH_KEYS, level)
  MD.TestSshLogin(key)
  MD.RemoveSshKey(key, MM.SSH_KEYS, level)
  MD.TestSshLogin(key, expect_fail=True)


def TestSshKeysWithSshKeys(level):
  ssh_key = MD.AddSshKey(MM.SSH_KEYS, store=False)
  ssh_key_legacy = MD.AddSshKey(MM.SSHKEYS_LEGACY)
  MD.TestSshLogin(ssh_key)
  MD.TestSshLogin(ssh_key_legacy)
  MD.RemoveSshKey(ssh_key, MM.SSH_KEYS, store=False)
  MD.RemoveSshKey(ssh_key_legacy, MM.SSHKEYS_LEGACY)
  MD.TestSshLogin(ssh_key, expect_fail=True)
  MD.TestSshLogin(ssh_key_legacy, expect_fail=True)


def TestSshKeysMixedProjectInstanceLevel():
  instance_key = MD.AddSshKey(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  project_key = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(project_key)
  MD.TestSshLogin(instance_key)
  MD.RemoveSshKey(instance_key, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.RemoveSshKey(project_key, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(project_key, expect_fail=True)
  MD.TestSshLogin(instance_key, expect_fail=True)


def TestSshKeysIgnoresProjectLevelKeys():
  ssh_key = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  ssh_key_legacy = MD.AddSshKey(MM.SSHKEYS_LEGACY, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(ssh_key, expect_fail=True)
  MD.TestSshLogin(ssh_key_legacy)
  MD.RemoveSshKey(ssh_key_legacy, MM.SSHKEYS_LEGACY, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(ssh_key_legacy, expect_fail=True)
  MD.TestSshLogin(ssh_key)
  MD.RemoveSshKey(ssh_key, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(ssh_key, expect_fail=True)


def TestBlockProjectSshKeysIgnoresProjectLevelKeys():
  SetBlockProjectSshKeys(True)
  project_key = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  instance_key = MD.AddSshKey(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(project_key, expect_fail=True)
  MD.TestSshLogin(instance_key)
  SetBlockProjectSshKeys(False)
  MD.TestSshLogin(project_key)
  MD.TestSshLogin(instance_key)
  MD.RemoveSshKey(instance_key, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.RemoveSshKey(project_key, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(project_key, expect_fail=True)
  MD.TestSshLogin(instance_key, expect_fail=True)


def main():
  global MD

  compute = utils.GetCompute(discovery, oauth2client.client.GoogleCredentials)
  testee = MM.FetchMetadataDefault('testee')
  MD = MM(compute, testee)
  MD.Define('enable-oslogin', False, MM.PROJECT_LEVEL)
  SetBlockProjectSshKeys(False)

  TestLoginSshKeys(MM.INSTANCE_LEVEL)
  TestLoginSshKeys(MM.PROJECT_LEVEL)
  TestSshKeysWithSshKeys(MM.INSTANCE_LEVEL)
  TestSshKeysWithSshKeys(MM.PROJECT_LEVEL)
  TestSshKeysMixedProjectInstanceLevel()
  TestSshKeysIgnoresProjectLevelKeys()
  TestBlockProjectSshKeysIgnoresProjectLevelKeys()


if __name__ == '__main__':
  utils.RunTest(main)
