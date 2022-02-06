#!/usr/bin/env python3
# Copyright 2022 Google LLC
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

"""Repack a tarball. ARM tar isn't valid for GCE Image creation.

Parameters (retrieved from instance metadata):

source_tarball: The GCS path to the tarball to download and repack.
dest_tarball: The GCS path to the resulting uploaded tarball.
"""

import logging
import tarfile

import utils


def main():
  # Get Parameters.
  source_tarball = utils.GetMetadataAttribute('source_tarball',
                                              raise_on_not_found=True)
  dest_tarball = utils.GetMetadataAttribute('dest_tarball',
                                            raise_on_not_found=True)

  logging.info('Downloading %s to /root.tar.gz.', source_tarball)
  cmd = ['gsutil', 'cp', source_tarball, '/root.tar.gz']
  utils.Execute(cmd)

  with tarfile.open('/root.tar.gz') as tar:
    tar.extractall()
  logging.info('Extracted /disk.raw')

  # Packs a gzipped tar file with disk.raw inside
  disk_tar_gz = '/root-repack.tar.gz'
  logging.info('(Re-)compressing it into %s', disk_tar_gz)
  tar = tarfile.open(disk_tar_gz, 'w:gz')
  tar.add('/disk.raw', arcname='disk.raw')
  tar.close()

  # Upload tar.
  logging.info('Saving %s to %s', disk_tar_gz, dest_tarball)
  utils.UploadFile(disk_tar_gz, dest_tarball)


if __name__ == '__main__':
  try:
    main()
    logging.success('Repackage was successful')
  except Exception as e:
    logging.error('Repackage failed: %s', e)
