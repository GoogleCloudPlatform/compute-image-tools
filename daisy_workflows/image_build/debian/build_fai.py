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

"""Build the Debian image on a GCE VM.

Parameters (retrieved from instance metadata):

debian_cloud_images_version: The debian-cloud-images scripts git commit ID
to use.
debian_version: The FAI tool debian version to be requested.
image_dest: The Cloud Storage destination for the resultant image.
"""

import logging
import os
import platform
import shutil
import subprocess
import tarfile
import urllib.request

import utils


# The 3.7 version of shutil.copytree doesn't support skipping existing
# directories. This code is simplified shutil.copytree from 3.9
def mycopytree(src, dst):
    with os.scandir(src) as itr:
        entries = list(itr)
    os.makedirs(dst, exist_ok=True)
    for srcentry in entries:
        dstname = os.path.join(dst, srcentry.name)
        if srcentry.is_dir():
            mycopytree(srcentry, dstname)
        else:
            shutil.copy2(srcentry, dstname)
    return dst


def main():
  # Get Parameters.
  build_date = utils.GetMetadataAttribute(
      'build_date', raise_on_not_found=True)
  debian_cloud_images_version = '69783f7417aefb332d5d7250ba242adeca444131'
  debian_version = utils.GetMetadataAttribute(
      'debian_version', raise_on_not_found=True)
  outs_path = utils.GetMetadataAttribute('daisy-outs-path',
                                         raise_on_not_found=True)

  logging.info('debian-cloud-images version: %s' % debian_cloud_images_version)
  logging.info('debian version: %s' % debian_version)

  # force an apt-get update before next install
  utils.AptGetInstall.first_run = True
  utils.AptGetInstall(['fai-server', 'fai-setup-storage'])

  # Download and setup debian's debian-cloud-images scripts.
  url_params = {
      'project': 'debian-cloud-images',
      'version': debian_cloud_images_version,
  }
  url_params['filename'] = '%(project)s-%(version)s' % url_params

  url = ('https://salsa.debian.org/cloud-team/'
         '%(project)s/-/archive/%(version)s/%(filename)s.tar.gz' % url_params)

  logging.info('Downloading %(project)s at version %(version)s', url_params)
  urllib.request.urlretrieve(url, 'fci.tar.gz')
  with tarfile.open('fci.tar.gz') as tar:
    tar.extractall()
  logging.info('Downloaded and extracted %s.', url)

  work_dir = url_params['filename']
  config_space = os.getcwd() + '/' + work_dir + '/config_space/'

  # We are going to replace this with our variant
  os.remove(config_space + 'class/BULLSEYE.var')

  # Remove failing test method for now.
  os.remove(config_space + 'hooks/tests.CLOUD')

  # Copy our classes to the FAI config space
  mycopytree('/files/fai_config', config_space)

  # Set scripts executable (daisy doesn't preserve this)
  os.chmod(config_space + 'scripts/GCE_CLEAN/10-gce-clean', 0o755)
  os.chmod(config_space + 'scripts/GCE_SPECIFIC/12-sshd', 0o755)

  # Config fai-tool
  # Base classes
  fai_classes = ['DEBIAN', 'CLOUD', 'GCE', 'GCE_SDK', 'LINUX_IMAGE_CLOUD',
                 'GCE_SPECIFIC', 'GCE_CLEAN']

  # Arch-specific classes
  if platform.machine() == 'aarch64':
    fai_classes += ['ARM64', 'GRUB_EFI_ARM64', 'BACKPORTS_LINUX']
  else:
    fai_classes += ['AMD64', 'GRUB_CLOUD_AMD64']

  # Version-specific classes
  if debian_version == 'buster':
    fai_classes += ['BUSTER']
  elif debian_version == 'bullseye':
    fai_classes += ['BULLSEYE']
  elif debian_version == 'sid':
    fai_classes += ['SID']

  image_size = '10G'
  disk_name = 'disk.raw'

  # Run fai-tool.
  cmd = ['fai-diskimage', '--verbose', '--hostname', 'debian', '--class',
         ','.join(fai_classes), '--size', image_size, '--cspace',
         config_space, disk_name]
  logging.info('Starting build in %s with params: %s', work_dir, ' '.join(cmd))
  returncode, output = utils.Execute(
      cmd, cwd=work_dir, capture_output=True, raise_errors=False)

  # Verbose printing to console for debugging.
  for line in output.splitlines():
    print(line)

  if returncode != 0:
    raise subprocess.CalledProcessError(returncode, cmd)

  # Packs a gzipped tar file with disk.raw inside
  disk_tar_gz = 'debian-{}-{}.tar.gz'.format(debian_version, build_date)
  logging.info('Compressing it into tarball %s', disk_tar_gz)
  tar = tarfile.open(disk_tar_gz, 'w:gz', format=tarfile.GNU_FORMAT)
  tar.add('%s/%s' % (work_dir, disk_name), arcname=disk_name)
  tar.close()

  # Upload tar.
  image_dest = os.path.join(outs_path, 'root.tar.gz')
  logging.info('Saving %s to %s', disk_tar_gz, image_dest)
  utils.UploadFile(disk_tar_gz, image_dest)


if __name__ == '__main__':
  try:
    main()
    logging.success('Debian build was successful!')
  except Exception as e:
    logging.error('Debian build failed: %s', e)
