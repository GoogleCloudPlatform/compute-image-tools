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

from google import auth
from googleapiclient import discovery
import utils

MM = utils.MetadataManager
MD = None


def SetBlockProjectSshKeys(state):
  MD.SetMetadata('block-project-ssh-keys', state, MM.INSTANCE_LEVEL)


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


def TestAdminLoginSshKeys(level):
  key = MD.AddSshKey(MM.SSH_KEYS, level)
  MD.TestSshLogin(key, as_root=True)
  MD.RemoveSshKey(key, MM.SSH_KEYS, level)
  MD.TestSshLogin(key, as_root=True, expect_fail=True)
  # Re-add a new ssh key and check if it still has root privileges.
  new_key = MD.AddSshKey(MM.SSH_KEYS, level)
  MD.TestSshLogin(new_key, as_root=True)
  MD.RemoveSshKey(new_key, MM.SSH_KEYS, level)
  MD.TestSshLogin(new_key, as_root=True, expect_fail=True)


def main():
  global MD

  credentials, _ = auth.default()
  compute = utils.GetCompute(discovery, credentials)
  testee = MM.FetchMetadataDefault('testee')
  MD = MM(compute, testee)
  MD.SetMetadata('enable-oslogin', False, MM.PROJECT_LEVEL)
  SetBlockProjectSshKeys(False)

  TestLoginSshKeys(MM.INSTANCE_LEVEL)
  TestLoginSshKeys(MM.PROJECT_LEVEL)
  TestAdminLoginSshKeys(MM.INSTANCE_LEVEL)
  TestAdminLoginSshKeys(MM.PROJECT_LEVEL)
  TestSshKeysWithSshKeys(MM.INSTANCE_LEVEL)
  TestSshKeysWithSshKeys(MM.PROJECT_LEVEL)
  TestSshKeysMixedProjectInstanceLevel()
  TestSshKeysIgnoresProjectLevelKeys()
  TestBlockProjectSshKeysIgnoresProjectLevelKeys()


if __name__ == '__main__':
  utils.RunTest(main)
