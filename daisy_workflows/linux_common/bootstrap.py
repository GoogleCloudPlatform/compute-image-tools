#!/usr/bin/env python2
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

"""Bootstrapper for running a VM script.

Args:
  files_gcs_dir: The Cloud Storage location containing the files.
    This dir will be used to run the 'script' requested by Metadata.
  script: The main script to be run
  prefix: a string prefix for outputing status
"""
import logging
import os
import subprocess
import urllib2


DIR = '/files'


def GetMetadataAttribute(attribute):
  url = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/%s' % attribute
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def DebianInstallGoogleApiPythonClient(prefix):
  logging.info('%sStatus: Installing google-api-python-client', prefix)
  subprocess.check_call(['apt-get', 'update'])
  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  cmd = ['apt-get', '-q', '-y', 'install', 'python-pip']
  subprocess.Popen(cmd, env=env).communicate()

  cmd = ['pip', 'install', '--upgrade', 'google-api-python-client']
  subprocess.check_call(cmd)


def Bootstrap():
  """Get files, run."""
  try:
    prefix = GetMetadataAttribute('prefix')
    status = prefix + 'Status'
    logging.info('%s: Starting bootstrap.py.', status)

    # Optional flag
    try:
      if GetMetadataAttribute('debian_install_google_api_python_client'):
        DebianInstallGoogleApiPythonClient(prefix)
    except urllib2.HTTPError:
      pass

    gcs_dir = GetMetadataAttribute('files_gcs_dir')
    script = GetMetadataAttribute('script')
    full_script = os.path.join(DIR, script)
    subprocess.check_call(['mkdir', DIR])
    subprocess.check_call(
        ['gsutil', '-m', 'cp', '-r', os.path.join(gcs_dir, '*'), DIR])
    logging.info('%s: Making script %s executable.', status, full_script)
    subprocess.check_call(['chmod', '+x', script], cwd=DIR)
    logging.info('%s: Running %s.', status, full_script)
    subprocess.check_call([full_script], cwd=DIR)
  except Exception as e:
    fail = prefix + 'Failed'
    print('%s: error: %s' % (fail, str(e)))


if __name__ == '__main__':
  logging.basicConfig(level=logging.DEBUG)
  Bootstrap()
