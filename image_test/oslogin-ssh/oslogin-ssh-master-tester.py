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

import datetime

from google import auth
from googleapiclient import discovery
import utils
from utils import logging

MM = utils.MetadataManager
MD = None
MASTER_KEY = None
OSLOGIN_TESTER = None
OSADMINLOGIN_TESTER = None
TESTEE = None
TESTER_SH = 'slave_tester.sh'
OSLOGIN_USERS_LIB = None
UNIQUE_ID_USER = None


def MasterExecuteInSsh(machine, commands, expect_fail=False):
  ret, output = utils.ExecuteInSsh(
      MASTER_KEY, MD.ssh_user, machine, commands, expect_fail,
      capture_output=True)
  output = output.strip() if output else None
  return ret, output


@utils.RetryOnFailure()
def MasterExecuteInSshRetry(machine, commands, expect_fail=False):
  return MasterExecuteInSsh(machine, commands, expect_fail)


def AddOsLoginKeys():
  _, key_oslogin = MasterExecuteInSsh(
      OSLOGIN_TESTER, [TESTER_SH, 'add_key'])
  _, key_osadminlogin = MasterExecuteInSsh(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'add_key'])
  return key_oslogin, key_osadminlogin


def RemoveOsLoginKeys():
  MasterExecuteInSsh(OSLOGIN_TESTER, [TESTER_SH, 'remove_key'])
  MasterExecuteInSsh(OSADMINLOGIN_TESTER, [TESTER_SH, 'remove_key'])


def SetEnableOsLogin(state, level, md=None):
  md = md if md else MD
  md.SetMetadata('enable-oslogin', state, level)


def GetServiceAccountUsername(machine):
  _, username = MasterExecuteInSsh(
      machine,
      ['gcloud', 'compute', 'os-login', 'describe-profile',
       '--format="value(posixAccounts.username)"'])
  return username


@utils.RetryOnFailure()
def CheckAuthorizedKeys(user, key, expect_empty=False):
  _, auth_keys = MasterExecuteInSsh(TESTEE, ['google_authorized_keys', user])
  auth_keys = auth_keys if auth_keys else ''
  if expect_empty and key in auth_keys:
    raise ValueError(
        'Os Login key DETECTED in google_authorized_keys when NOT expected')
  elif not expect_empty and key not in auth_keys:
    raise ValueError(
        'Os Login key NOT DETECTED in google_authorized_keys when expected')


@utils.RetryOnFailure()
def CheckNss(user_oslogin, user_osadminlogin, expect_empty=False):
  _, users = MasterExecuteInSsh(TESTEE, ['getent', 'passwd'])
  if expect_empty and (user_oslogin in users or user_osadminlogin in users):
    raise ValueError(
        'Os Login usernames DETECTED in getent passwd (nss) when NOT expected')
  elif not expect_empty and (
      user_oslogin not in users or user_osadminlogin not in users):
    raise ValueError(
        'Os Login usernames NOT DETECTED in getent passwd (nss) when expected')


def TestLoginFromSlaves(user_oslogin, user_osadminlogin, expect_fail=False):
  host_oslogin = '%s@%s' % (user_oslogin, TESTEE)
  host_osadminlogin = '%s@%s' % (user_osadminlogin, TESTEE)
  MasterExecuteInSshRetry(
      OSLOGIN_TESTER, [TESTER_SH, 'test_login', host_oslogin],
      expect_fail=expect_fail)
  MasterExecuteInSshRetry(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'test_login', host_osadminlogin],
      expect_fail=expect_fail)
  MasterExecuteInSshRetry(
      OSLOGIN_TESTER, [TESTER_SH, 'test_login_sudo', host_oslogin],
      expect_fail=True)
  MasterExecuteInSshRetry(
      OSADMINLOGIN_TESTER, [TESTER_SH, 'test_login_sudo', host_osadminlogin],
      expect_fail=expect_fail)


def TestOsLogin(level):
  key_oslogin, key_osadminlogin = AddOsLoginKeys()
  user_oslogin = GetServiceAccountUsername(OSLOGIN_TESTER)
  user_osadminlogin = GetServiceAccountUsername(OSADMINLOGIN_TESTER)
  SetEnableOsLogin(True, level)
  CheckNss(user_oslogin, user_osadminlogin)
  CheckAuthorizedKeys(user_oslogin, key_oslogin)
  CheckAuthorizedKeys(user_osadminlogin, key_osadminlogin)
  TestLoginFromSlaves(user_oslogin, user_osadminlogin)
  RemoveOsLoginKeys()
  TestLoginFromSlaves(user_oslogin, user_osadminlogin, expect_fail=True)
  key_oslogin, key_osadminlogin = AddOsLoginKeys()
  TestLoginFromSlaves(user_oslogin, user_osadminlogin)
  SetEnableOsLogin(None, level)
  TestLoginFromSlaves(user_oslogin, user_osadminlogin, expect_fail=True)
  CheckNss(user_oslogin, user_osadminlogin, expect_empty=True)
  CheckAuthorizedKeys(user_oslogin, key_oslogin, expect_empty=True)
  CheckAuthorizedKeys(user_osadminlogin, key_osadminlogin, expect_empty=True)
  RemoveOsLoginKeys()


def TestMetadataWithOsLogin(level):
  tester_key = MD.AddSshKey(MM.SSH_KEYS, level)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(True, level)
  MD.TestSshLogin(tester_key, expect_fail=True)
  SetEnableOsLogin(None, level)
  MD.TestSshLogin(tester_key)
  MD.RemoveSshKey(tester_key, MM.SSH_KEYS, level)
  MD.TestSshLogin(tester_key, expect_fail=True)


def TestOsLoginFalseInInstance():
  tester_key = MD.AddSshKey(MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(True, MM.PROJECT_LEVEL)
  MD.TestSshLogin(tester_key, expect_fail=True)
  SetEnableOsLogin(False, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key)
  SetEnableOsLogin(None, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key, expect_fail=True)
  SetEnableOsLogin(None, MM.PROJECT_LEVEL)
  MD.TestSshLogin(tester_key)
  MD.RemoveSshKey(tester_key, MM.SSH_KEYS, MM.INSTANCE_LEVEL)
  MD.TestSshLogin(tester_key, expect_fail=True)


def AddKeyOsLogin(key):
  """Returns the key fingerprint for using later to remove it"""
  key = open(key).read()

  # Microseconds since epoch + 30 minutes from now
  expire_obj = datetime.datetime.now() + datetime.timedelta(minutes=30)
  expire_str = '%d' % (int(expire_obj.strftime("%s")) * 1000000)

  body = {'key': key, 'expirationTimeUsec': expire_str}
  request = OSLOGIN_USERS_LIB.importSshPublicKey(parent=UNIQUE_ID_USER,
                                                 body=body)
  response = request.execute()
  keys = response[u'loginProfile'][u'sshPublicKeys']

  # retrieves the fingerprint of the added key
  for k_dict in keys.values():
    if k_dict[u'key'] == key:
      return k_dict[u'fingerprint']

  raise Exception('Added key not found on existing keys: %s' % key)


def RemoveKeyOsLogin(fingerprint):
  name = '%s/sshPublicKeys/%s' % (UNIQUE_ID_USER, fingerprint)
  OSLOGIN_USERS_LIB.sshPublicKeys().delete(name=name).execute()


def main():
  global MD
  global MASTER_KEY
  global OSLOGIN_TESTER
  global OSADMINLOGIN_TESTER
  global TESTEE
  global OSLOGIN_USERS_LIB
  global UNIQUE_ID_USER

  logging.info("master: starting environment configuration")

  TESTEE = MM.FetchMetadataDefault('testee')
  OSLOGIN_TESTER = MM.FetchMetadataDefault('osLoginTester')
  OSADMINLOGIN_TESTER = MM.FetchMetadataDefault('osAdminLoginTester')
  credentials, _ = auth.default()
  OSLOGIN_USERS_LIB = utils.GetOslogin(discovery, credentials).users()
  UNIQUE_ID_USER = utils.GetServiceAccountUniqueIDUser()
  username = utils.GetCurrentLoginProfileUsername(OSLOGIN_USERS_LIB,
                                                  UNIQUE_ID_USER)
  compute = utils.GetCompute(discovery, credentials)

  MD = MM(compute, TESTEE, username)
  SetEnableOsLogin(None, MM.PROJECT_LEVEL)
  SetEnableOsLogin(None, MM.INSTANCE_LEVEL)

  # Enable OsLogin in slaves
  md = MM(compute, OSLOGIN_TESTER, username)
  SetEnableOsLogin(True, MM.INSTANCE_LEVEL, md)
  md = MM(compute, OSADMINLOGIN_TESTER, username)
  SetEnableOsLogin(True, MM.INSTANCE_LEVEL, md)

  # Add key in Metadata and in OsLogin to allow access peers in both modes
  MASTER_KEY = MD.AddSshKey(MM.SSH_KEYS, MM.PROJECT_LEVEL)
  master_key_fingerprint = AddKeyOsLogin(MASTER_KEY + '.pub')

  # Execute tests
  try:
    logging.info("master: starting: TestOsLogin(MM.INSTANCE_LEVEL)")
    TestOsLogin(MM.INSTANCE_LEVEL)

    logging.info("master: starting: TestOsLogin(MM.PROJECT_LEVEL)")
    TestOsLogin(MM.PROJECT_LEVEL)

    logging.info(
        "master: starting: TestMetadataWithOsLogin(MM.INSTANCE_LEVEL)")
    TestMetadataWithOsLogin(MM.INSTANCE_LEVEL)

    logging.info("master: starting: TestMetadataWithOsLogin(MM.PROJECT_LEVEL)")
    TestMetadataWithOsLogin(MM.PROJECT_LEVEL)

    logging.info("master: starting test: TestOsLoginFalseInInstance()")
    TestOsLoginFalseInInstance()

  # Clean keys
  finally:
    logging.info("master: starting environment cleanup")
    MD.RemoveSshKey(MASTER_KEY, MM.SSH_KEYS, MM.PROJECT_LEVEL)
    RemoveKeyOsLogin(master_key_fingerprint)


if __name__ == '__main__':
  utils.RunTest(main)
