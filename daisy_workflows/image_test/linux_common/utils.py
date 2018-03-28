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
    key, user, machine, cmds, expect_fail=False, capture_output=False):
  """Execute commands through ssh.

  Args:
    key: string, the path of the private key to use in the ssh connection.
    user: string, the user used to connect through ssh.
    machine: string, the hostname of the machine to connect.
    cmds: list[string], the commands to be execute in the ssh session.
    expect_fail: bool, indicates if the failure in the execution is expected.
    capture_output: bool, indicates if the output of the command should be
        captured.

  Returns:
    ret, out if capture_output=True.
  """
  ssh_command = [
      'ssh', '-i', key, '-o', 'IdentitiesOnly=yes', '-o', 'ConnectTimeout=10',
      '-o', 'StrictHostKeyChecking=no', '-o', 'UserKnownHostsFile=/dev/null',
      '%s@%s' % (user, machine)
  ]
  ret, out = Execute(
      ssh_command + cmds, raise_errors=False, capture_output=capture_output)
  if expect_fail and ret == 0:
    raise ValueError('SSH command succeeded when expected to fail')
  elif not expect_fail and ret != 0:
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
  """Configure Logging system."""
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
    self.last_fingerprint = None
    self.ssh_user = ssh_user
    self.md_items = {}
    md_obj = self._FetchMetadata(self.INSTANCE_LEVEL)
    self.md_items[self.INSTANCE_LEVEL] = \
        md_obj['items'] if 'items' in md_obj else []
    md_obj = self._FetchMetadata(self.PROJECT_LEVEL)
    self.md_items[self.PROJECT_LEVEL] = \
        md_obj['items'] if 'items' in md_obj else []

  def _FetchMetadata(self, level):
    """Fetch metadata from the server.

    Args:
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.
    """
    if level == self.PROJECT_LEVEL:
      request = self.compute.projects().get(project=self.project)
      md_id = 'commonInstanceMetadata'
    else:
      request = self.compute.instances().get(
          project=self.project, zone=self.zone, instance=self.instance)
      md_id = 'metadata'
    response = request.execute()
    return response[md_id]

  @RetryOnFailure
  def StoreMetadata(self, level):
    """Store Metadata.

    Args:
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to store the metadata.
    """
    md_obj = self._FetchMetadata(level)
    md_obj['items'] = self.md_items[level]
    if level == self.PROJECT_LEVEL:
      request = self.compute.projects().setCommonInstanceMetadata(
          project=self.project, body=md_obj)
    else:
      request = self.compute.instances().setMetadata(
          project=self.project, zone=self.zone, instance=self.instance,
          body=md_obj)
    request.execute()

  def ExtractKeyItem(self, md_key, level):
    """Extract a given key value from the metadata.

    Args:
      md_key: string, the key of the metadata value to be searched.
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.

    Returns:
      md_item: dict, in the format {'key', md_key, 'value', md_value}.
      None: if md_key was not found.
    """
    for md_item in self.md_items[level]:
        if md_item['key'] == md_key:
            return md_item

  def Define(self, md_key, md_value, level, store=True):
    """Add or update a metadata key with a new value in a given level.

    Args:
      md_key: string, the key of the metadata.
      md_value: string, value to be added or updated.
      level: enum, INSTANCE_LEVEL or PROJECT_LEVEL to fetch the metadata.
      store: bool, if True, saves metadata to GCE server.
    """
    if not level:
        level = self.INSTANCE_LEVEL
    md_item = self.ExtractKeyItem(md_key, level)
    if md_item and md_value is None:
      self.md_items[level].remove(md_item)
    elif not md_item:
      md_item = {'key': md_key, 'value': md_value}
      self.md_items[level].append(md_item)
    else:
      md_item['value'] = md_value
    if store:
        self.StoreMetadata(level)

  def AddSshKey(self, md_key, level=None, store=True):
    """Generate and add an ssh key to the metadata

    Args:
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      level: enum, INSTANCE_LEVEL (default) or PROJECT_LEVEL to fetch the
          metadata.
      store: bool, if True, saves metadata to GCE server.

    Returns:
      key_name: string, the name of the file with the generated private key.
    """
    if not level:
        level = self.INSTANCE_LEVEL
    key, key_name = GenSshKey(self.ssh_user)
    md_item = self.ExtractKeyItem(md_key, level)
    if not md_item:
      md_item = {'key': md_key, 'value': key}
      self.md_items[level].append(md_item)
    else:
      md_item['value'] = '\n'.join([md_item['value'], key])
    if store:
        self.StoreMetadata(level)
    return key_name

  def RemoveSshKey(self, key, md_key, level=None, store=True):
    """Remove an ssh key to the metadata

    Args:
      key: string, the key to be removed.
      md_key: string, SSH_KEYS or SSHKEYS_LEGACY, defines where to add the key.
      level: enum, INSTANCE_LEVEL (default) or PROJECT_LEVEL to fetch the
          metadata.
      store: bool, if True, saves metadata to GCE server.
    """
    if not level:
        level = self.INSTANCE_LEVEL
    md_item = self.ExtractKeyItem(md_key, level)
    md_item['value'] = re.sub('.*%s.*\n?' % key, '', md_item['value'])
    if not md_item['value']:
      self.md_items[level].remove(md_item)
    if store:
        self.StoreMetadata(level)

  @RetryOnFailure
  def TestSshLogin(self, key, expect_fail=False):
    """Try to login to self.instance using key.

    Args:
      key: string, the private key to be used in the ssh connection.
    """
    ExecuteInSsh(
        key, self.ssh_user, self.instance, ['echo', 'Logged'],
        expect_fail=expect_fail)

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
