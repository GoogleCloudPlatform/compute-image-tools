#!/usr/bin/env python3
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

debian_cloud_images_version: The debian-cloud-images scripts git commit ID
to use.
debian_version: The FAI tool debian version to be requested.
image_dest: The Cloud Storage destination for the resultant image.
google_cloud_repo: The repository branch to use for packages.cloud.google.com.
"""

import collections
import json
import logging
import os
import tarfile
import urllib.request

import utils


def main():
  # Get Parameters.
  build_date = utils.GetMetadataAttribute(
      'build_date', raise_on_not_found=True)
  debian_cloud_images_version = utils.GetMetadataAttribute(
      'debian_cloud_images_version', raise_on_not_found=True)
  debian_version = utils.GetMetadataAttribute(
      'debian_version', raise_on_not_found=True)
  google_cloud_repo = utils.GetMetadataAttribute(
      'google_cloud_repo', raise_on_not_found=True)
  image_dest = utils.GetMetadataAttribute('image_dest',
                                          raise_on_not_found=True)
  outs_path = utils.GetMetadataAttribute('daisy-outs-path',
                                         raise_on_not_found=True)

  logging.info('debian-cloud-images version: %s' % debian_cloud_images_version)
  logging.info('debian version: %s' % debian_version)

  # force an apt-get update before next install
  utils.AptGetInstall.first_run = True
  utils.AptGetInstall(['apt-transport-https', 'qemu-utils', 'dosfstools'])

  debian_host_version = utils.Execute(['cat', '/etc/debian_version'],
                                      capture_output=True)
  # the FAI's version in stretch does not satisfy our need, so the version from
  # stretch-backports is needed.
  if debian_host_version[1].startswith('9'):
    utils.AptGetInstall(
        ['fai-server', 'fai-setup-storage'], 'stretch-backports')
  else:
    utils.AptGetInstall(['fai-server', 'fai-setup-storage'])

  # Download and setup debian's debian-cloud-images scripts.
  url_params = {
      'project': 'debian-cloud-images',
      'version': debian_cloud_images_version,
  }
  url_params['filename'] = '%(project)s-%(version)s' % url_params

  url = ('https://salsa.debian.org/cloud-team/'
         '%(project)s/-/archive/%(version)s/%(filename)s.tar.gz' % url_params)
  logging.info('Downloading %(project)s at version %(version)s' % url_params)
  urllib.request.urlretrieve(url, 'fci.tar.gz')
  with tarfile.open('fci.tar.gz') as tar:
    tar.extractall()
  logging.info('Downloaded and extracted %s.' % url)

  # Config fai-tool
  work_dir = url_params['filename']
  fai_classes = ['DEBIAN', 'CLOUD', 'GCE', 'GCE_SDK', 'AMD64',
                 'GRUB_CLOUD_AMD64', 'LINUX_IMAGE_CLOUD']
  if debian_version == 'stretch':
    fai_classes += ['STRETCH', 'BACKPORTS', 'BACKPORTS_LINUX']
  elif debian_version == 'buster':
    fai_classes += ['BUSTER']
  elif debian_version == 'sid':
    fai_classes += ['SID']
  image_size = '10G'
  disk_name = 'disk.raw'
  config_space = os.getcwd() + work_dir + '/config_space/'

  # Copy GCE_SPECIFIC fai classes.
  utils.Execute(['cp', '/files/fai_config/packages/GCE_SPECIFIC',
                 config_space + 'package_config/GCE_SPECIFIC'])
  os.mkdir(config_space + 'files/etc/apt/sources.list.d/gce.list')
  utils.Execute(
      ['cp', '/files/fai_config/sources/GCE_SPECIFIC',
       config_space + 'files/etc/apt/sources.list.d/gce.list/GCE_SPECIFIC'])
  utils.Execute(
      ['cp', '/files/fai_config/sources/file_modes',
       config_space + 'files/etc/apt/sources.list.d/gce.list/file_modes'])
  utils.Execute(
      ['cp', '/files/fai_config/sources/repository.GCE_SPECIFIC',
       config_space + 'hooks/repository.GCE_SPECIFIC'])
  fai_classes += ['GCE_SPECIFIC']

  # GCE staging package repo.
  if google_cloud_repo == 'staging' or google_cloud_repo == 'unstable':
    os.mkdir(config_space + 'files/etc/apt/sources.list.d/gce_staging.list')
    utils.Execute(
        ['cp', '/files/fai_config/sources/GCE_STAGING',
         config_space
         + 'files/etc/apt/sources.list.d/gce_staging.list/GCE_STAGING'])
    utils.Execute(
        ['cp', '/files/fai_config/sources/file_modes',
         config_space
         + 'files/etc/apt/sources.list.d/gce_staging.list/file_modes'])
    utils.Execute(
        ['cp', '/files/fai_config/sources/repository.GCE_STAGING',
         config_space + 'hooks/repository.GCE_STAGING'])
    fai_classes += ['GCE_STAGING']

  # GCE unstable package repo.
  if google_cloud_repo == 'unstable':
    os.mkdir(config_space + 'files/etc/apt/sources.list.d/gce_unstable.list')
    utils.Execute(
        ['cp', '/files/fai_config/sources/GCE_UNSTABLE',
         config_space
         + 'files/etc/apt/sources.list.d/gce_unstable.list/GCE_UNSTABLE'])
    utils.Execute(
        ['cp', '/files/fai_config/sources/file_modes',
         config_space
         + 'files/etc/apt/sources.list.d/gce_unstable.list/file_modes'])
    utils.Execute(
        ['cp', '/files/fai_config/sources/repository.GCE_UNSTABLE',
         config_space + 'hooks/repository.GCE_UNSTABLE'])
    fai_classes += ['GCE_UNSTABLE']

  # Run fai-tool.
  cmd = ['fai-diskimage', '--verbose', '--hostname', 'debian', '--class',
         ','.join(fai_classes), '--size', image_size, '--cspace',
         config_space, disk_name]
  logging.info('Starting build in %s with params: %s' % (
      work_dir, ' '.join(cmd)))
  utils.Execute(cmd, cwd=work_dir, capture_output=True)

  # Packs a gzipped tar file with disk.raw inside
  disk_tar_gz = 'debian-{}-{}.tar.gz'.format(debian_version, build_date)
  logging.info('Compressing it into tarball %s' % disk_tar_gz)
  tar = tarfile.open(disk_tar_gz, 'w:gz')
  tar.add('%s/disk.raw' % work_dir, arcname='disk.raw')
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
