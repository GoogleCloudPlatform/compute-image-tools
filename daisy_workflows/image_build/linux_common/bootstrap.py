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
build-files-gcs-dir: The Cloud Storage location containing the build files.
  This dir of build files must contain a build.py containing the build logic.
"""
import logging
import os
import subprocess
import sys
import urllib2


BUILD_DIR = '/build_files'


def GetMetadataAttribute(attribute):
  url = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/%s' % attribute
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def ExecuteScript(script):
  """Runs a script and logs the output."""
  process = subprocess.Popen(script, shell=True, executable='/bin/bash',
                             cwd=BUILD_DIR, stderr=subprocess.STDOUT,
                             stdout=subprocess.PIPE)
  while True:
    for line in iter(process.stdout.readline, b''):
      message = line.decode('utf-8', 'replace').rstrip('\n')
      if message:
        logging.info(message)
    if process.poll() is not None:
      break
  logging.info('BuildStatus: %s: return code %s', script, process.returncode)


def Bootstrap():
  """Get build files, run build, poweroff."""
  try:
    logging.info('BuildStatus: Starting bootstrap.py.')
    build_gcs_dir = GetMetadataAttribute('build_files_gcs_dir')
    build_script = GetMetadataAttribute('build_script')
    full_build_script = os.path.join(BUILD_DIR, build_script)
    subprocess.call(['mkdir', BUILD_DIR])
    subprocess.call(
        ['gsutil', '-m', 'cp', '-r', os.path.join(build_gcs_dir, '*'), BUILD_DIR])
    logging.info('BuildStatus: Making build script %s executable.', full_build_script)
    subprocess.call(['chmod', '+x', build_script], cwd=BUILD_DIR)
    logging.info('BuildStatus: Running %s.', full_build_script)
    ExecuteScript(full_build_script)
  except:
    logging.error('BuildFailed: Cannot run %s.', full_build_script)


if __name__ == '__main__':
  logger = logging.getLogger()
  logger.setLevel(logging.DEBUG)
  stdout = logging.StreamHandler(sys.stdout)
  stdout.setLevel(logging.DEBUG)
  logger.addHandler(stdout)
  Bootstrap()
