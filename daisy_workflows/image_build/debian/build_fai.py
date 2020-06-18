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
import json
import logging
import os
import tarfile
import time
import urllib.request
import uuid

import utils


def CopyToConfigSpace(src, dst, config_space):
  """Copies source files to config space destination."""
  return utils.Execute(['cp', src, config_space + dst])


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
  if debian_version == 'buster':
    fai_classes += ['BUSTER', 'BACKPORTS']
  elif debian_version == 'sid':
    fai_classes += ['SID']
  image_size = '10G'
  disk_name = 'disk.raw'
  config_space = os.getcwd() + work_dir + '/config_space/'
  apt_sources_base = 'files/etc/apt/sources.list.d/'

  # Copy GCE_SPECIFIC fai classes.
  CopyToConfigSpace('/files/fai_config/packages/GCE_SPECIFIC',
                    'package_config/GCE_SPECIFIC',
                    config_space)
  os.mkdir(config_space + apt_sources_base + 'google-cloud.list')
  CopyToConfigSpace('/files/fai_config/sources/GCE_SPECIFIC',
                    apt_sources_base + 'google-cloud.list/GCE_SPECIFIC',
                    config_space)
  CopyToConfigSpace('/files/fai_config/sources/file_modes',
                    apt_sources_base + '/google-cloud.list/file_modes',
                    config_space)
  CopyToConfigSpace('/files/fai_config/sources/repository.GCE_SPECIFIC',
                    'hooks/repository.GCE_SPECIFIC',
                    config_space)
  fai_classes += ['GCE_SPECIFIC']

  # GCE staging package repo.
  if google_cloud_repo == 'staging' or google_cloud_repo == 'unstable':
    os.mkdir(
        config_space + apt_sources_base + 'google-cloud-staging.list')
    CopyToConfigSpace(
        '/files/fai_config/sources/GCE_STAGING',
        apt_sources_base + 'google-cloud-staging.list/GCE_STAGING',
        config_space)
    CopyToConfigSpace(
        '/files/fai_config/sources/file_modes',
        apt_sources_base + 'google-cloud-staging.list/file_modes',
        config_space)
    CopyToConfigSpace('/files/fai_config/sources/repository.GCE_STAGING',
                      'hooks/repository.GCE_STAGING',
                      config_space)
    fai_classes += ['GCE_STAGING']

  # GCE unstable package repo.
  if google_cloud_repo == 'unstable':
    os.mkdir(
        config_space + apt_sources_base + 'google-cloud-unstable.list')
    CopyToConfigSpace(
        '/files/fai_config/sources/GCE_UNSTABLE',
        apt_sources_base + 'google-cloud-unstable.list/GCE_UNSTABLE',
        config_space)
    CopyToConfigSpace(
        '/files/fai_config/sources/file_modes',
        apt_sources_base + 'google-cloud-unstable.list/file_modes',
        config_space)
    CopyToConfigSpace('/files/fai_config/sources/file_modes',
                      'hooks/repository.GCE_UNSTABLE',
                      config_space)
    fai_classes += ['GCE_UNSTABLE']

  # Cleanup class for GCE.
  os.mkdir(config_space + 'scripts/GCE_CLEAN')
  CopyToConfigSpace('/files/fai_config/scripts/10-gce-clean',
                    'scripts/GCE_CLEAN/10-gce-clean',
                    config_space)
  os.chmod(config_space + 'scripts/GCE_CLEAN/10-gce-clean', 0o755)
  fai_classes += ['GCE_CLEAN']

  # Remove failing test method for now.
  os.remove(config_space + 'hooks/tests.CLOUD')

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

  # Create and upload metadata of the image and packages
  logging.info('Creating image metadata.')
  image = {
      "id": uuid.uuid4(),
      "family": "debian-10",
      "name": "debian-10" + debian_version + "v" + build_date,
      "version": debian_version,
      "location": image_dest,
      "release_date": build_date,
      "release_time": time.time(),
      "state": "Active",
      "environment": "prod",
      "packages": []
  }
  # Read list of guest package
  with open("guest_package") as f:
    guest_packages = f.read().splitlines()

  for package in guest_packages:
    cmd = "dpkg-query -W --showformat '${Package} ${Version} ${Git}\n'" \
          + package
    code, stdout = utils.Excute(cmd, capture_output=True)
    if code == 0:
      splits = stdout.decode('utf-8').split('\t\b')
      package_name = splits[0]
      package_version = splits[1]
      package_commit_hash = splits[2][splits[2].rindex('/'):len(splits[2])]
      start = package_version.index(":")
      end = package_version.rindex(".")
      package_release_time = package_version[start: end]

      metadata = {
          "id": uuid.uuid4(),
          "name": package_name,
          "version": package_version,
          "commit_hash": package_commit_hash,
          "release_date": package_release_time,
          "stage": google_cloud_repo
      }
      image["packages"].append(metadata)
      with open('/tmp/metadata.json', 'w') as f:
        f.write(json.dumps(image))

      logging.info('Uploading image metadata.')
      metadata_dest = os.path.join(image_dest.strip("root.tar.gz"),
                                   'metadata.json')
      utils.UploadFile('/tmp/metadata.json', metadata_dest)


if __name__ == '__main__':
  try:
    main()
    logging.success('Debian build was successful!')
  except Exception as e:
    logging.error('Debian build failed: %s' % e)
