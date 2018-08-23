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

"""Build the Debian image on a GCE VM.

Parameters (retrieved from instance metadata):

fai_cloud_images_version: The debian's fai-cloud-images scripts git commit ID
to use.
debian_version: The FAI tool debian version to be requested.
image_dest: The Cloud Storage destination for the resultant image.
"""

import collections
import json
import logging
import os
import tarfile
import urllib

import utils

# The following package is necessary to retrieve from https repositories.
utils.AptGetInstall(['apt-transport-https', 'qemu-utils'])


def main():
  # Get Parameters.
  fai_cloud_images_version = utils.GetMetadataAttribute(
      'fai_cloud_images_version', raise_on_not_found=True)
  debian_version = utils.GetMetadataAttribute(
      'debian_version', raise_on_not_found=True)
  image_dest = utils.GetMetadataAttribute('image_dest',
      raise_on_not_found=True)
  outs_path = utils.GetMetadataAttribute('daisy-outs-path',
      raise_on_not_found=True)

  logging.info('fai-cloud-images version: %s' % fai_cloud_images_version)
  logging.info('debian version: %s' % debian_version)

  # First, install fai-client from fai-project repository
  key_url = 'https://fai-project.org/download/2BF8D9FE074BCDE4.asc'
  urllib.urlretrieve(key_url, 'key.asc')
  utils.Execute(['apt-key', 'add', 'key.asc'])
  with open('/etc/apt/sources.list.d/fai-project.list', 'w') as fai_list:
    fai_list.write('deb https://fai-project.org/download stretch koeln')

  # force an apt-get update before next install
  utils.AptGetInstall.first_run = True
  utils.AptGetInstall(['fai-server', 'fai-setup-storage'])

  # Download and setup debian's fai-cloud-images scripts.
  url_params = {
      'project': 'fai-cloud-images',
      'commit': fai_cloud_images_version,
  }
  url_params['filename'] = '%(project)s-%(commit)s' % url_params

  url = "https://salsa.debian.org/cloud-team/" + \
      "%(project)s/-/archive/%(commit)s/%(filename)s.tar.gz" % url_params
  logging.info('Downloading fai-cloud-image at commit %s' %
               fai_cloud_images_version)
  urllib.urlretrieve(url, 'fci.tar.gz')
  with tarfile.open('fci.tar.gz') as tar:
    tar.extractall()
  logging.info('Downloaded and extracted %s.' % url)

  # Run fai-tool.
  work_dir = url_params['filename']
  fai_bin = 'bin/build'
  cmd = [fai_bin, debian_version, 'gce', 'amd64', 'disk']
  logging.info('Starting build in %s with params: %s' % (
      work_dir, ' '.join(cmd))
  )
  utils.Execute(cmd, cwd=work_dir, capture_output=True)

  # Packs a gzipped tar file with disk.raw inside
  disk_tar_gz = 'disk.tar.gz'
  logging.info('Compressing it into tarball %s' % disk_tar_gz)
  tar = tarfile.open(disk_tar_gz, "w:gz")
  tar.add('%s/disk.raw' % work_dir, arcname="disk.raw")
  tar.close()

  # Upload tar.
  logging.info('Saving %s to %s' % (disk_tar_gz, image_dest))
  utils.UploadFile(disk_tar_gz, image_dest)

  # Create and upload the synopsis of the image.
  logging.info('Creating image synopsis.')
  synopsis = {}
  packages = collections.OrderedDict()
  _, output = utils.Execute(['dpkg-query', '-W'], capture_output=True)
  for line in output.split('\n')[:-1]:  # Last line is an empty line.
    parts = line.split()
    packages[parts[0]] = parts[1]
  synopsis['installed_packages'] = packages
  with open('/tmp/synopsis.json', 'w') as f:
    f.write(json.dumps(synopsis))
  logging.info('Uploading image synopsis.')
  synopsis_dest = os.path.join(outs_path, 'synopsis.json')
  utils.UploadFile('/tmp/synopsis.json', synopsis_dest)


if __name__ == '__main__':
  try:
    main()
    logging.success('Debian build was successful!')
  except Exception as e:
    logging.error('Debian build failed: %s' % e)
