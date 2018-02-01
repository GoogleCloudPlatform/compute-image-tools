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


import logging
import re
import time
import uuid

import utils

utils.AptGetInstall(['python-pip'])
utils.Execute(['pip', 'install', '--upgrade', 'google-api-python-client'])

from googleapiclient import discovery
from oauth2client.client import GoogleCredentials

TESTEE = None
PROJECT = None
ZONE = None
COMPUTE = None
SSH_KEYS = 'ssh-keys'
SSHKEYS = 'sshKeys'
INSTANCE_LEVEL = 1
PROJECT_LEVEL = 2


def gen_ssh_key():
  key_name = 'daisy-test-key-' + str(uuid.uuid4())
  utils.Execute(
      ['ssh-keygen', '-t', 'rsa', '-N', '', '-f', key_name, '-C', key_name])
  with open(key_name + '.pub', 'r') as original: data = original.read()
  return "tester:" + data, key_name


def get_metadata(level):
  if level == PROJECT_LEVEL:
    request = COMPUTE.projects().get(project=PROJECT)
    md_id = 'commonInstanceMetadata'
  else:
    request = COMPUTE.instances().get(
        project=PROJECT, zone=ZONE, instance=TESTEE)
    md_id = 'metadata'
  response = request.execute()
  return response[md_id]


def set_metadata(md_obj, level):
  if level == PROJECT_LEVEL:
    request = COMPUTE.projects().setCommonInstanceMetadata(
        project=PROJECT, body=md_obj)
  else:
    request = COMPUTE.instances().setMetadata(
        project=PROJECT, zone=ZONE, instance=TESTEE, body=md_obj)
  response = request.execute()


def extract_key_item(md_obj, md_key):
  try:
    md_item = (md for md in md_obj['items'] if md['key'] == md_key).next()
  except StopIteration:
    md_item = None
  return md_item


def add_key(md_obj, md_key):
  key, key_name = gen_ssh_key()
  md_item = extract_key_item(md_obj, md_key)
  if not md_item:
    md_item = {'key': md_key, 'value': key}
    md_obj['items'].append(md_item)
  else:
    md_item['value'] = '\n'.join([md_item['value'], key])
  return key_name


def remove_key(md_obj, md_key, key):
  md_item = extract_key_item(md_obj, md_key)
  md_item['value'] = re.sub('.*%s.*\n?' % key, '', md_item['value'])
  if not md_item['value']:
    md_obj['items'].remove(md_item)


def add_key_single(md_key, level):
  md_obj = get_metadata(level)
  key = add_key(md_obj, md_key)
  set_metadata(md_obj, level)
  return key


def remove_key_single(key, md_key, level):
  md_obj = get_metadata(level)
  remove_key(md_obj, md_key, key)
  set_metadata(md_obj, level)


def test_login(key, expect_fail=False):
  for try_again in range(3):
    ret, _ = utils.Execute(
        ['ssh', '-i', key, '-o', 'StrictHostKeyChecking=no', '-o',
        'UserKnownHostsFile=/dev/null', 'tester@' + TESTEE, 'echo', 'Logged'],
        raise_errors=False)
    if expect_fail and ret == 0:
      error = 'SSH Loging succeeded when expected to fail'
    elif not expect_fail and ret != 0:
      error = 'SSH Loging failed when expected to succeed'
    else:
      return
    time.sleep(5)
  raise ValueError(error)


def set_block_project_ssh_keys(state):
  md_obj = get_metadata(INSTANCE_LEVEL)
  md_key = 'block-project-ssh-keys'
  md_item = extract_key_item(md_obj, md_key)
  if not md_item:
    md_item = {'key': md_key, 'value': state}
    md_obj['items'].append(md_item)
  else:
    md_item['value'] = state
  set_metadata(md_obj, INSTANCE_LEVEL)


def test_login_ssh_keys(level):
  key = add_key_single(SSH_KEYS, level)
  test_login(key)
  remove_key_single(key, SSH_KEYS, level)
  test_login(key, expect_fail=True)


def test_ssh_keys_with_sshKeys(level):
  md_obj = get_metadata(level)
  ssh_keys_key = add_key(md_obj, SSH_KEYS)
  sshKey_key = add_key(md_obj, SSHKEYS)
  set_metadata(md_obj, level)
  test_login(ssh_keys_key)
  test_login(sshKey_key)
  md_obj = get_metadata(level)
  remove_key(md_obj, SSH_KEYS, ssh_keys_key)
  remove_key(md_obj, SSHKEYS, sshKey_key)
  set_metadata(md_obj, level)
  test_login(ssh_keys_key, expect_fail=True)
  test_login(sshKey_key, expect_fail=True)


def test_ssh_keys_mixed_project_instance_level():
  i_key = add_key_single(SSH_KEYS, INSTANCE_LEVEL)
  p_key = add_key_single(SSH_KEYS, PROJECT_LEVEL)
  test_login(p_key)
  test_login(i_key)
  remove_key_single(i_key, SSH_KEYS, INSTANCE_LEVEL)
  remove_key_single(p_key, SSH_KEYS, PROJECT_LEVEL)
  test_login(p_key, expect_fail=True)
  test_login(i_key, expect_fail=True)


def test_sshKeys_ignores_project_level_keys():
  ssh_keys_key = add_key_single(SSH_KEYS, PROJECT_LEVEL)
  sshKey_key = add_key_single(SSHKEYS, INSTANCE_LEVEL)
  test_login(ssh_keys_key, expect_fail=True)
  test_login(sshKey_key)
  remove_key_single(sshKey_key, SSHKEYS, INSTANCE_LEVEL)
  test_login(sshKey_key, expect_fail=True)
  test_login(ssh_keys_key)
  remove_key_single(ssh_keys_key, SSH_KEYS, PROJECT_LEVEL)
  test_login(ssh_keys_key, expect_fail=True)


def test_block_project_ssh_keys_ignores_project_level_keys():
  set_block_project_ssh_keys(True)
  p_key = add_key_single(SSH_KEYS, PROJECT_LEVEL)
  i_key = add_key_single(SSH_KEYS, INSTANCE_LEVEL)
  test_login(p_key, expect_fail=True)
  test_login(i_key)
  set_block_project_ssh_keys(False)
  test_login(p_key)
  test_login(i_key)
  remove_key_single(i_key, SSH_KEYS, INSTANCE_LEVEL)
  remove_key_single(p_key, SSH_KEYS, PROJECT_LEVEL)
  test_login(p_key, expect_fail=True)
  test_login(i_key, expect_fail=True)


def main():
  global COMPUTE
  global ZONE
  global PROJECT
  global TESTEE

  COMPUTE = utils.GetCompute(discovery, GoogleCredentials) 
  ZONE = utils.GetMetadataParam('zone')
  PROJECT = utils.GetMetadataParam('project')
  TESTEE = utils.GetMetadataParam('testee')

  test_login_ssh_keys(INSTANCE_LEVEL)
  test_login_ssh_keys(PROJECT_LEVEL)
  test_ssh_keys_with_sshKeys(INSTANCE_LEVEL)
  test_ssh_keys_with_sshKeys(PROJECT_LEVEL)
  test_ssh_keys_mixed_project_instance_level()
  test_sshKeys_ignores_project_level_keys()
  test_block_project_ssh_keys_ignores_project_level_keys()


if __name__=='__main__':
  utils.RunTest(main) 
