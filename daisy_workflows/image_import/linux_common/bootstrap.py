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
import-files-gcs-dir: The Cloud Storage location containing the import files.
  This dir of import files must contain a import.py containing the import logic.
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
  """Get import files, run import."""
  try:
    logging.info('Starting bootstrap.py.')
    import_gcs_dir = GetMetadataAttribute('import_files_gcs_dir')
    import_script = GetMetadataAttribute('import_script')
    import_dir = '/import_files'
    full_import_script = os.path.join(import_dir, import_script)
    subprocess.check_call(['mkdir', import_dir])
    subprocess.check_call(
        ['gsutil', '-m', 'cp', '-r', os.path.join(import_gcs_dir, '*'),
        import_dir])
    logging.info('Making import script %s executable.', full_import_script)
    subprocess.check_call(['chmod', '+x', import_script], cwd=import_dir)
    logging.info('Running %s.', full_import_script)
    subprocess.check_call([full_import_script], cwd=import_dir)
  except Exception as e:
    logging.error('TranslateFailed: error: %s', str(e))

if __name__ == '__main__':
  logging.basicConfig(level=logging.DEBUG)
  Bootstrap()
