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
from oauth2client.client import GoogleCredentials
from utils import MetadataManager as MM

MD = None


def SetBlockProjectSshKeys(state):
  MD.Define('block-project-ssh-keys', state, MM.INSTANCE_LEVEL)


def TestLoginSshKeys(level):
  key = MD.AddSshKey(MM.SSH_KEYS, level)
  MD.TestSshLogin(key)
  MD.RemoveSshKey(key, MM.SSH_KEYS, level)
  MD.TestSshLogin(key, expectFail=True)


def TestSshKeysWithSshKeys(level):
  MD.FetchMetadata(level)
  ssh_keysKey = MD.AddSshKey(MM.SSH_KEYS, store=False)
  sshKeysLegacyKey = MD.AddSshKey(MM.SSHKEYS_LEGACY, store=False)
  MD.StoreMetadata()
  MD.TestSshLogin(ssh_keysKey)
  MD.TestSshLogin(sshKeysLegacyKey)
  MD.FetchMetadata(level)
  MD.RemoveSshKey(ssh_keysKey, MM.SSH_KEYS, store=False)
  MD.RemoveSshKey(shKeysLegacyKey, MM.SSHKEYS_LEGACY, store=False)
  MD.StoreMetadata()
  MD.TestSshLogin(ssh_keysKey, expectFail=True)
  MD.TestSshLogin(sshKeysLegacyKey, expectFail=True)


def TestSshKeysMixedProjectInstanceLevel():
  iKey = MD.AddSshKey(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  pKey = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(pKey)
  MD.TestSshLogin(iKey)
  MD.RemoveSshKey(iKey, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.RemoveSshKey(pKey, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(pKey, expectFail=True)
  MD.TestSshLogin(iKey, expectFail=True)


def TestSshKeysIgnoresProjectLevelKeys():
  ssh_keysKey = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  sshKeysLegacyKey = MD.AddSshKey(MM.SSHKEYS_LEGACY, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(ssh_keysKey, expectFail=True)
  MD.TestSshLogin(sshKeysLegacyKey)
  MD.RemoveSshKey(sshKeysLegacyKey, MM.SSHKEYS_LEGACY, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(sshKeysLegacyKey, expectFail=True)
  MD.TestSshLogin(ssh_keysKey)
  MD.RemoveSshKey(ssh_keysKey, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(ssh_keysKey, expectFail=True)


def TestBlockProjectSshKeysIgnoresProjectLevelKeys():
  SetBlockProjectSshKeys(True)
  pKey = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  iKey = MD.AddSshKey(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(pKey, expectFail=True)
  MD.TestSshLogin(iKey)
  SetBlockProjectSshKeys(False)
  MD.TestSshLogin(pKey)
  MD.TestSshLogin(iKey)
  MD.RemoveSshKey(iKey, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.RemoveSshKey(pKey, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  MD.TestSshLogin(pKey, expectFail=True)
  MD.TestSshLogin(iKey, expectFail=True)


def main():
  global MD

  compute = utils.GetCompute(discovery, GoogleCredentials)
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
