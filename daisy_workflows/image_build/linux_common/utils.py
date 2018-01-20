#!/usr/bin/python
# Copyright 2017 Google Inc. All Rights Reserved.
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

import logging
import os
import stat
import subprocess
import sys
import urllib2


def Fail(message):
  logging.error('BuildFailed: %s', message)


def Status(message):
  logging.info('BuildStatus: %s', message)


def Success(message):
  logging.info('BuildSuccess: %s', message)


def YumInstall(package_list):
  if YumInstall.first_run:
    Execute(['yum', 'update'])
    YumInstall.first_run = False
  Execute(['yum', '-y', 'install'] + package_list)
YumInstall.first_run = True


def AptGetInstall(package_list):
  if AptGetInstall.first_run:
    try:
      Execute(['apt-get', 'update'])
    except subprocess.CalledProcessError as error:
      Status('Apt update failed, trying again: %s' % error)
      Execute(['apt-get', 'update'], raise_errors=False)
    AptGetInstall.first_run = False

  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  return Execute(['apt-get', '-q', '-y', 'install'] + package_list, env=env)
AptGetInstall.first_run = True


def PipInstall(package_list):
  """Install Python modules via pip. Assumes pip is already installed."""
  return Execute(['pip', 'install', '-U'] + package_list)


def Gsutil(params):
  """Call gsutil."""
  env = os.environ.copy()
  return Execute(['gsutil', '-m'] + params, capture_output=True, env=env)


def Execute(cmd, cwd=None, capture_output=False, env=None, raise_errors=True):
  """Execute an external command (wrapper for Python subprocess)."""
  Status('Executing command: %s' % str(cmd))
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
      Status('Command returned error status %d' % returncode)
  if output:
    Status(output)
  return returncode, output, None


def HttpGet(url, headers=None):
  request = urllib2.Request(url)
  if headers:
    for key in headers.keys():
      request.add_unredirected_header(key, headers[key])
  return urllib2.urlopen(request).read()


def GetMetadataParam(name, default_value=None, raise_on_not_found=False):
  try:
    url = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/%s' % name
    return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
  except urllib2.HTTPError:
    if raise_on_not_found:
      raise ValueError('Metadata key "%s" not found' % name)
    else:
      return default_value


def GetMetadataParamBool(name, default_value):
  value = GetMetadataParam(name, default_value)
  if not value:
    return False
  return True if value.lower() == 'yes' else False


def MakeExecutable(file_path):
  os.chmod(file_path, os.stat(file_path).st_mode | stat.S_IEXEC)


def ReadFile(file_path, strip=False):
  content = open(file_path).read()
  if strip:
    return content.strip()
  return content


def WriteFile(file_path, content, mode='w'):
  with open(file_path, mode) as fp:
    fp.write(content)


def SetupLogging():
  logger = logging.getLogger()
  logger.setLevel(logging.DEBUG)
  stdout = logging.StreamHandler(sys.stdout)
  stdout.setLevel(logging.DEBUG)
  logger.addHandler(stdout)


SetupLogging()
