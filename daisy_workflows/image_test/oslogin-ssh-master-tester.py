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
MASTER_KEY = None
OSLOGIN_TESTER = None
OSADMINLOGIN_TESTER = None
TESTEE = None
TESTER_SH = 'slave_tester.sh'


def MasterExecuteInSsh(machine, commands, expectFail=False):
  ret, output = utils.ExecuteInSsh(
      MASTER_KEY, MD.ssh_user, machine, commands, expectFail,
      capture_output=True)
  output = output.strip() if output else None
  return ret, output


@utils.RetryOnFailure
def MasterExecuteInSshRetry(machine, commands, expectFail=False):
  return MasterExecuteInSsh(machine, commands, expectFail)


def AddOsLoginKeys():
  _, keyOsLogin = MasterExecuteInSsh(
      OSLOGIN_TESTER, [TESTER_SH, 'add_key'])
  _, keyOsAdminLogin = MasterExecuteInSsh(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'add_key'])
  return keyOsLogin, keyOsAdminLogin


def RemoveOsLoginKeys():
  MasterExecuteInSsh(OSLOGIN_TESTER, [TESTER_SH, 'remove_key'])
  MasterExecuteInSsh(OSADMINLOGIN_TESTER, [TESTER_SH, 'remove_key'])


def SetEnableOsLogin(state, level, md=None):
  md = md if md else MD
  md.DefineSingle('enable-oslogin', state, level)


def GetServiceAccountUsername(machine):
  _, username = MasterExecuteInSsh(
      machine,
      ['gcloud', 'compute', 'os-login', 'describe-profile',
      '--format="value(posixAccounts.username)"'])
  return username


@utils.RetryOnFailure
def CheckAuthorizedKeys(user, key, expectEmpty=False):
  _, authKeys = MasterExecuteInSsh(TESTEE, ['google_authorized_keys', user])
  authKeys = authKeys if authKeys else ''
  if expectEmpty and key in authKeys:
    raise ValueError(
        'OsLogin key DETECTED in google_authorized_keys when NOT expected')
  elif not expectEmpty and key not in authKeys:
    raise ValueError(
        'OsLogin key NOT DETECTED in google_authorized_keys when expected')


@utils.RetryOnFailure
def CheckNss(userOsLogin, userOsAdminLogin, expectEmpty=False):
  _, users = MasterExecuteInSsh(TESTEE, ['getent', 'passwd'])
  if expectEmpty and (userOsLogin in users or userOsAdminLogin in users):
    raise ValueError(
        'OsLogin usernames DETECTED in getend passwd (nss) when NOT expected')
  elif not expectEmpty and (userOsLogin not in users or
      userOsAdminLogin not in users):
    raise ValueError(
        'OsLogin usernames NOT DETECTED in getend passwd (nss) when expected')


def TestLoginFromSlaves(userOsLogin, userOsAdminLogin, expectFail=False):
  hostOsLogin = '%s@%s' % (userOsLogin, TESTEE)
  hostOsAdminLogin = '%s@%s' % (userOsAdminLogin, TESTEE)
  MasterExecuteInSshRetry(
      OSLOGIN_TESTER, [TESTER_SH, 'test_login', hostOsLogin],
      expectFail=expectFail)
  MasterExecuteInSshRetry(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'test_login', hostOsAdminLogin],
      expectFail=expectFail)
  MasterExecuteInSshRetry(
      OSLOGIN_TESTER, [TESTER_SH, 'test_login_sudo', hostOsLogin],
      expectFail=True)
  MasterExecuteInSshRetry(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'test_login_sudo', hostOsAdminLogin],
      expectFail=expectFail)


def TestOsLogin(level):
  keyOsLogin, keyOsAdminLogin = AddOsLoginKeys()
  userOsLogin = GetServiceAccountUsername(OSLOGIN_TESTER)
  userOsAdminLogin = GetServiceAccountUsername(OSADMINLOGIN_TESTER)
  SetEnableOsLogin(True, level)
  CheckNss(userOsLogin, userOsAdminLogin)
  CheckAuthorizedKeys(userOsLogin, keyOsLogin)
  CheckAuthorizedKeys(userOsAdminLogin, keyOsAdminLogin)
  TestLoginFromSlaves(userOsLogin, userOsAdminLogin)
  RemoveOsLoginKeys()
  TestLoginFromSlaves(userOsLogin, userOsAdminLogin, expectFail=True)
  keyOsLogin, keyOsAdminLogin = AddOsLoginKeys()
  TestLoginFromSlaves(userOsLogin, userOsAdminLogin)
  SetEnableOsLogin(None, level)
  TestLoginFromSlaves(userOsLogin, userOsAdminLogin, expectFail=True)
  CheckNss(userOsLogin, userOsAdminLogin, expectEmpty=True)
  CheckAuthorizedKeys(userOsLogin, keyOsLogin, expectEmpty=True)
  CheckAuthorizedKeys(userOsAdminLogin, keyOsAdminLogin, expectEmpty=True)
  RemoveOsLoginKeys()


def TestMetadataWithOsLogin(level):
  tester_key = MD.AddSshKeySingle(MM.SSH_KEYS, level)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(True, level)
  MD.TestSshLogin(tester_key, expectFail=True)
  SetEnableOsLogin(None, level)
  MD.TestSshLogin(tester_key)
  MD.RemoveSshKeySingle(tester_key, MM.SSH_KEYS, level)
  MD.TestSshLogin(tester_key, expectFail=True)


def TestOsLoginFalseInInstance():
  tester_key = MD.AddSshKeySingle(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(True, MM.PROJECT_LEVEL)
  MD.TestSshLogin(tester_key, expectFail=True)
  SetEnableOsLogin(False, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(None, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key, expectFail=True)
  SetEnableOsLogin(None, MM.PROJECT_LEVEL)
  MD.TestSshLogin(tester_key)
  MD.RemoveSshKeySingle(tester_key, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key, expectFail=True)


def GetCurrentUsername():
  # TODO: replace gcloud usage by python CLI
  _, username = utils.Execute(
      ['gcloud', 'compute', 'os-login', 'describe-profile',
      '--format', 'value(posixAccounts.username)'], capture_output=True)
  return username.strip()


def AddKeyOsLogin(key):
  # TODO: replace gcloud usage by python CLI
  utils.Execute(
      ['gcloud', 'compute', 'os-login', 'ssh-keys', 'add', '--key-file', key])


def RemoveKeyOsLogin(key):
  # TODO: replace gcloud usage by python CLI
  utils.Execute(
      ['gcloud', 'compute', 'os-login', 'ssh-keys', 'remove', '--key-file',
      key])


def main():
  global MD
  global MASTER_KEY
  global OSLOGIN_TESTER
  global OSADMINLOGIN_TESTER
  global TESTEE

  TESTEE = MM.FetchMetadataDefault('testee')
  OSLOGIN_TESTER = MM.FetchMetadataDefault('osLoginTester')
  OSADMINLOGIN_TESTER = MM.FetchMetadataDefault('osAdminLoginTester')
  username = GetCurrentUsername()
  compute = utils.GetCompute(discovery, GoogleCredentials)
  MD = MM(compute, TESTEE, username)
  SetEnableOsLogin(None, MM.PROJECT_LEVEL)
  SetEnableOsLogin(None, MM.INSTANCE_LEVEL)

  # Enable OsLogin in slaves
  md = MM(compute, OSLOGIN_TESTER, username)
  SetEnableOsLogin(True, MM.INSTANCE_LEVEL, md)
  md = MM(compute, OSADMINLOGIN_TESTER, username)
  SetEnableOsLogin(True, MM.INSTANCE_LEVEL, md)

  # Add key in Metadata and in OsLogin to allow access peers in both modes
  MASTER_KEY = MD.AddSshKeySingle(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  AddKeyOsLogin(MASTER_KEY + '.pub')

  # Execute tests
  TestOsLogin(MM.INSTANCE_LEVEL)
  TestOsLogin(MM.PROJECT_LEVEL)
  TestMetadataWithOsLogin(MM.INSTANCE_LEVEL)
  TestMetadataWithOsLogin(MM.PROJECT_LEVEL)
  TestOsLoginFalseInInstance()

  # Clean keys
  MD.RemoveSshKeySingle(MASTER_KEY, MM.SSH_KEYS, MM.PROJECT_LEVEL)
  RemoveKeyOsLogin(MASTER_KEY + '.pub')


if __name__ == '__main__':
  utils.RunTest(main)
