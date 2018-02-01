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

import logging
import os
import subprocess
import sys
import time
import trace
import traceback
import urllib2


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


def GetMetadataParam(name, default_value=None, raise_on_not_found=False):
  try:
    url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % name
    return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
  except urllib2.HTTPError:
    if raise_on_not_found:
      raise ValueError('Metadata key "%s" not found' % name)
    else:
      return default_value


def GetCompute(discovery, GoogleCredentials):
  credentials = GoogleCredentials.get_application_default()
  compute = discovery.build('compute', 'v1', credentials=credentials)
  return compute


def RunTest(test_func):
  try:
    tracer = trace.Trace(
        ignoredirs=[sys.prefix, sys.exec_prefix], trace=1, count=0)
    tracer.runfunc(test_func)
    print('TestSuccess: Test finished.')
  except Exception as e:
    print('TestFailed: error: ')
    print(str(e))
    traceback.print_exc()


def SetupLogging():
  logging_level = logging.DEBUG
  logging.basicConfig(level=logging_level)
  console = logging.StreamHandler()
  console.setLevel(logging_level)
  logging.getLogger().addHandler(console)

SetupLogging()
