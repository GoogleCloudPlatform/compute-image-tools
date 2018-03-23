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

"""Utility functions for all VM scripts."""

import functools
import logging
import os
import re
import subprocess
import sys
import time
import trace
import traceback
import urllib2
import uuid


def AptGetInstall(package_list):
  if AptGetInstall.first_run:
    try:
      Execute(['apt-get', 'update'])
    except subprocess.CalledProcessError as error:
      logging.warning('Apt update failed, trying again: %s', error)
      Execute(['apt-get', 'update'], raise_errors=False)
    AptGetInstall.first_run = False

  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  return Execute(['apt-get', '-q', '-y', 'install'] + package_list, env=env)


AptGetInstall.first_run = True


def Execute(cmd, cwd=None, capture_output=False, env=None, raise_errors=True):
  """Execute an external command (wrapper for Python subprocess)."""
  logging.info('Command: %s', str(cmd))
  returncode = 0
  output = None
  try:
    if capture_output:
      output = subprocess.check_output(cmd, cwd=cwd, env=env)
    else:
      subprocess.check_call(cmd, cwd=cwd, env=env)
  except subprocess.CalledProcessError as e:
    if raise_errors:
      raise
    else:
      returncode = e.returncode
      output = e.output
      logging.exception('Command returned error status %d', returncode)
  if output:
    logging.info(output)
  return returncode, output


def HttpGet(url, headers=None):
  request = urllib2.Request(url)
  if headers:
    for key in headers.keys():
      request.add_unredirected_header(key, headers[key])
  return urllib2.urlopen(request).read()


def GenSshKey(user):
  """Generate ssh key for user.

  Args:
    user: string, the user to create the ssh key for.

  Returns:
    ret, out if capture_output=True.
  """
  key_name = 'daisy-test-key-' + str(uuid.uuid4())
  Execute(
      ['ssh-keygen', '-t', 'rsa', '-N', '', '-f', key_name, '-C', key_name])
  with open(key_name + '.pub', 'r') as original:
    data = original.read().strip()
  return "%s:%s" % (user, data), key_name


def RetryOnFailure(func):
  """Function decorator to retry on an exception."""

  @functools.wraps(func)
  def Wrapper(*args, **kwargs):
    ratio = 1.5
    wait = 3
    ntries = 0
    start_time = time.time()
    end_time = start_time + 15 * 60
    while time.time() < end_time:
      ntries += 1
      try:
        response = func(*args, **kwargs)
      except Exception as e:
        logging.info(str(e))
        logging.info(
            'Function %s failed, waiting %d seconds, retrying %d ...',
            str(func), wait, ntries)
        time.sleep(wait)
        wait = wait * ratio
      else:
        logging.info(
            'Function %s executed in less then %d sec, with %d tentative(s)',
            str(func), time.time() - start_time, ntries)
        return response
    raise
  return Wrapper


def ExecuteInSsh(
    key, user, machine, cmds, expectFail=False, capture_output=False):
  """Execute commands through ssh.

  Args:
    key: string, the path of the private key to use in the ssh connection.
    user: string, the user used to connect through ssh.
    machine: string, the hostname of the machine to connect.
    cmds: list[string], the commands to be execute in the ssh session
    expectFail: bool, indicates if the failure in the execution is expected
    capture_output: bool, indicates if the output of the command should be
        captured

  Returns:
    ret, out if capture_output=True.
  """
  ssh_command = [
      'ssh', '-i', key, '-o', 'IdentitiesOnly=yes',
      '-o', 'StrictHostKeyChecking=no', '-o', 'UserKnownHostsFile=/dev/null',
      '%s@%s' % (user, machine)
  ]
  ret, out = Execute(
      ssh_command + cmds, raise_errors=False, capture_output=capture_output)
  if expectFail and ret == 0:
    raise ValueError('SSH command succeeded when expected to fail')
  elif not expectFail and ret != 0:
    raise ValueError('SSH command failed when expected to succeed')
  else:
    return ret, out


def GetCompute(discovery, GoogleCredentials):
  """Get google compute api cli object.

  Args:
    discovery: object, from googleapiclient.
    GoogleCredentials: object, from oauth2client.client.

  Returns:
    compute: object, the google compute api object.
  """
  credentials = GoogleCredentials.get_application_default()
  compute = discovery.build('compute', 'v1', credentials=credentials)
  return compute


def RunTest(test_func):
  """Run main test function and print TestSuccess or TestFailed.

  Args:
    test_func: function, the function to be tested.
  """
  try:
    tracer = trace.Trace(
        ignoredirs=[sys.prefix, sys.exec_prefix], trace=1, count=0)
    tracer.runfunc(test_func)
    print('TestSuccess: Test finished.')
  except Exception as e:
    print('TestFailed: error: ' + str(e))
    traceback.print_exc()


def SetupLogging():
  """Configure Loggin system."""
  logging_level = logging.DEBUG
  logging.basicConfig(level=logging_level)
  console = logging.StreamHandler()
  console.setLevel(logging_level)
  logging.getLogger().addHandler(console)


SetupLogging()


class MetadataManager:
  """Utilities to manage metadata."""

  SSH_KEYS = 'ssh-keys'
  SSHKEYS_LEGACY = 'sshKeys'
  INSTANCE_LEVEL = 1
  PROJECT_LEVEL = 2

  def __init__(self, compute, instance, ssh_user='tester'):
    """Constructor.

    Args:
      compute: object, from GetCompute.
      instance: string, the instance to manage the metadata.
      user: string, the user to create ssh keys and perform ssh tests.
    """
    self.zone = self.FetchMetadataDefault('zone')
    self.project = self.FetchMetadataDefault('project')
    self.compute = compute
    self.instance = instance
    self.md_obj = None
    self.level = None
    self.last_fingerprint = None
    self.ssh_user = ssh_user

  def FetchMetadata(self, level):
    """Fetch metadata from the server. This also fetches the fingerprint
    required for StoreMetadata.

    Args:
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.
    """
    self.level = level
    if level == self.PROJECT_LEVEL:
      request = self.compute.projects().get(project=self.project)
      md_id = 'commonInstanceMetadata'
    else:
      request = self.compute.instances().get(
          project=self.project, zone=self.zone, instance=self.instance)
      md_id = 'metadata'

    @RetryOnFailure
    def _FetchMetadata():
      response = request.execute()
      self.md_obj = response[md_id]
      if self.md_obj['fingerprint'] != self.last_fingerprint:
        self.last_fingerprint = self.md_obj['fingerprint']
      else:
        raise ValueError('FetchMetadata retrieved same fingerprint')
    _FetchMetadata()

  def StoreMetadata(self):
    """Store Metadata previously fetched with FetchMetadata."""
    if self.level == self.PROJECT_LEVEL:
      request = self.compute.projects().setCommonInstanceMetadata(
          project=self.project, body=self.md_obj)
    else:
      request = self.compute.instances().setMetadata(
          project=self.project, zone=self.zone, instance=self.instance,
          body=self.md_obj)
    request.execute()
    self.md_obj = None
    self.level = None

  def ExtractKeyItem(self, md_key):
    """Extract a given key value from the metadata previously fetched
    with FetchMetadata.

    Args:
      md_key: string, the key of the metadata value to be searched.

    Returns:
      md_item: dict, in the format {'key', md_key, 'value', md_value}.
      None: if md_key was not found.
    """
    try:
      md_item = (
          md for md in self.md_obj['items'] if md['key'] == md_key).next()
    except StopIteration:
      md_item = None
    return md_item

  def Define(self, md_key, md_value, level, store=True):
    """Add or update a metadata key with a new value in a given level.

    Args:
      md_key: string, the key of the metadata value to be added or updated.
      md_key: string, the key of the metadata value to be added or updated.
      store: bool, If true, this function already performs FetchMetadata and
          StoreMetadata.
    """
    if store:
        self.FetchMetadata(level)
    md_item = self.ExtractKeyItem(md_key)
    if md_item and md_value is None:
      self.md_obj['items'].remove(md_item)
    elif not md_item:
      md_item = {'key': md_key, 'value': md_value}
      self.md_obj['items'].append(md_item)
    else:
      md_item['value'] = md_value
    if store:
        self.StoreMetadata()

  def AddSshKey(self, md_key, level=None, store=True):
    """Generate and add an ssh key to the metadata previously fetched
        with FetchMetadata.

    Args:
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      store: bool, If true, this function already performs FetchMetadata and
          StoreMetadata.

    Returns:
      key_name: string, the name of the file with the generated private key.
    """
    if store:
        self.FetchMetadata(level)
    key, key_name = GenSshKey(self.ssh_user)
    md_item = self.ExtractKeyItem(md_key)
    if not md_item:
      md_item = {'key': md_key, 'value': key}
      self.md_obj['items'].append(md_item)
    else:
      md_item['value'] = '\n'.join([md_item['value'], key])
    if store:
        self.StoreMetadata()
    return key_name

  def RemoveSshKey(self, key, md_key, level=None, store=True):
    """Remove an ssh key to the metadata previously fetched
        with FetchMetadata.

    Args:
      key: string, the key to be removed.
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      store: bool, If true, this function already performs FetchMetadata and
          StoreMetadata.
    """
    if store:
        self.FetchMetadata(level)
    md_item = self.ExtractKeyItem(md_key)
    md_item['value'] = re.sub('.*%s.*\n?' % key, '', md_item['value'])
    if not md_item['value']:
      self.md_obj['items'].remove(md_item)
    if store:
        self.StoreMetadata()

  @RetryOnFailure
  def TestSshLogin(self, key, expectFail=False):
    """Try to login to self.instance using key.

    Args:
      key: string, the private key to be used in the ssh connection.
    """
    ExecuteInSsh(
        key, self.ssh_user, self.instance, ['echo', 'Logged'],
        expectFail=expectFail)

  @classmethod
  def FetchMetadataDefault(cls, name):
    """Fetch Metadata from default metadata server (local machine).

    Args:
      name: string, the metadata key to be fetched.

    Returns:
      value: the metadata value.
    """
    try:
      url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % name
      return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
    except urllib2.HTTPError:
      raise ValueError('Metadata key "%s" not found' % name)
