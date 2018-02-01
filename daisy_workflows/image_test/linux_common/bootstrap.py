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

"""Bootstrapper for running a VM script.

Args:
test-files-gcs-dir: The Cloud Storage location containing the test files.
  This dir of test files must contain a test.py containing the test logic.
"""
import base64
import logging
import os
import subprocess
import urllib2


def GetMetadataAttribute(attribute):
  url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % attribute
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def Bootstrap():
  """Get test files, run test."""
  try:
    logging.info('Starting bootstrap.py.')
    test_gcs_dir = GetMetadataAttribute('test_files_gcs_dir')
    test_script = GetMetadataAttribute('test_script')
    test_dir = '/test_files'
    full_test_script = os.path.join(test_dir, test_script)
    subprocess.check_call(['mkdir', test_dir])
    subprocess.check_call(
        ['gsutil', '-m', 'cp', '-r', os.path.join(test_gcs_dir, '*'), 
        test_dir])
    logging.info('Making test script %s executable.', full_test_script)
    subprocess.check_call(['chmod', '+x', test_script], cwd=test_dir)
    logging.info('Running %s.', full_test_script)
    subprocess.check_call([full_test_script], cwd=test_dir)
  except Exception as e:
    print('TestFailed: error: ')
    print(str(e))

if __name__ == '__main__':
  logging.basicConfig(level=logging.DEBUG)
  Bootstrap()
