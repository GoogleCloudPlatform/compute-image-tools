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

"""Saves the build logs and synopsis files to GCS from an EL install."""

import logging
import os

import utils

utils.AptGetInstall(['python-pip'])
utils.PipInstall(['google-cloud-storage'])


def main():
  raise_on_not_found = True
  logs_path = utils.GetMetadataAttribute('daisy-logs-path', raise_on_not_found)
  outs_path = utils.GetMetadataAttribute('daisy-outs-path', raise_on_not_found)

  # Mount the installer disk.
  utils.Execute(['mount', '-t', 'ext4', '/dev/sdb1', '/mnt'])

  logging.info('Installer root: %s' % os.listdir('/mnt'))
  logging.info('Build logs: %s' % os.listdir('/mnt/build-logs'))

  utils.UploadFile('/mnt/ks.cfg', '%s/' % logs_path)
  directory = '/mnt/build-logs'
  for f in os.listdir(directory):
    utils.UploadFile('%s/%s' % (directory, f), '%s/' % logs_path)
  utils.UploadFile('/mnt/build-logs/synopsis.json',
      '%s/synopsis.json' % outs_path)

  utils.Execute(['umount', '-l', '/mnt'])


if __name__ == '__main__':
  try:
    main()
    logging.success('Build logs successfully saved.')
  except Exception as e:
    logging.error('Failed to save build logs: %s.' % e)
