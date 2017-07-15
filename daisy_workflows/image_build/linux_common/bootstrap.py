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
import base64
import logging
import os
import subprocess
import urllib2
import zipfile


def GetMetadataAttribute(attribute):
  url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % attribute
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def Bootstrap():
  """Get build files, run build, poweroff."""
  try:
    logging.info('Starting bootstrap.py.')
    build_gcs_dir = GetMetadataAttribute('build_files_gcs_dir')
    build_script = GetMetadataAttribute('build_script')
    build_dir = '/build_files'
    full_build_script = os.path.join(build_dir, build_script)
    subprocess.call(['mkdir', build_dir])
    subprocess.call(
        ['gsutil', 'cp', '-r', os.path.join(build_gcs_dir, '*'), build_dir])
    logging.info('Making build script %s executable.', full_build_script)
    subprocess.call(['chmod', '+x', build_script], cwd=build_dir)
    logging.info('Running %s.', full_build_script)
    subprocess.call([full_build_script], cwd=build_dir)
  finally:
    os.system('sync')
    os.system('shutdown now -h')

if __name__ == '__main__':
  Bootstrap()
