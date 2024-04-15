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
  debian_cloud_images_version = '732707480b4c6a5174066a9d4d5a7b0c05ab0fb1'
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
  config_space = (os.getcwd() + '/' + work_dir + '/config_space/'
                  + debian_version + '/')

  # Remove upstream test cases that won't work here.
  os.remove(config_space + 'hooks/tests.BASE')

  # Copy our classes to the FAI config space
  mycopytree('/files/fai_config', config_space)

  # Set scripts executable (daisy doesn't preserve this)
  os.chmod(config_space + 'scripts/BOOKWORM/10-clean', 0o755)
  os.chmod(config_space + 'scripts/GCE_CLEAN/10-gce-clean', 0o755)
  os.chmod(config_space + 'scripts/GCE_SPECIFIC/12-sshd', 0o755)
  os.chmod(config_space + 'hooks/repository.BUSTER', 0o755)
  os.chmod(config_space + 'hooks/repository.GCE_SPECIFIC', 0o755)
  os.chmod(config_space + 'hooks/configure.GCE_SPECIFIC', 0o755)

  # Config fai-tool
  # Base classes used for everything
  fai_classes = ['BASE', 'DEBIAN', 'CLOUD', 'GCE', 'EXTRAS', 'IPV6_DHCP',
                 'GCE_SPECIFIC', 'GCE_CLEAN', 'LINUX_VARIANT_CLOUD']

  # Debian switched to systemd-timesyncd for ntp starting with bookworm
  if debian_version == 'buster' or debian_version == 'bullseye':
      fai_classes += ['TIME_CHRONY']
  else:
      fai_classes += ['TIME_SYSTEMD']

  # Arch-specific classes
  if platform.machine() == 'aarch64':
    if debian_version == 'buster' or debian_version == 'bullseye':
        fai_classes += ['ARM64_NO_SECURE_BOOT']
    else:
        fai_classes += ['ARM64_SECURE_BOOT']
    fai_classes += ['ARM64', 'GRUB_EFI_ARM64']
  else:
    fai_classes += ['AMD64', 'GRUB_CLOUD_AMD64']

  # Version-specific classes used to select release and kernel
  if debian_version == 'buster':  # Debian 10
    fai_classes += ['BUSTER', 'LINUX_VERSION_BASE+LINUX_VARIANT_CLOUD']
  elif debian_version == 'bullseye':  # Debian 11
    fai_classes += ['BULLSEYE', 'LINUX_VERSION_BASE+LINUX_VARIANT_CLOUD']
    # Use the backports kernel for Bullseye arm64 due to gVNIC.
    if platform.machine() == 'aarch64':  # Debian 11 arm64
      fai_classes += ['LINUX_VERSION_BACKPORTS',
                      'LINUX_VERSION_BACKPORTS+LINUX_VARIANT_CLOUD']
  elif debian_version == 'bookworm':  # Debian 12
    fai_classes += ['BOOKWORM', 'LINUX_VERSION_BASE+LINUX_VARIANT_CLOUD']
  elif debian_version == 'sid':  # Debian unstable
    fai_classes += ['SID', 'LINUX_VERSION_BASE+LINUX_VARIANT_CLOUD']

  image_size = '10G'
  disk_name = 'disk.raw'

  # Run fai-tool.
  cmd = ['fai-diskimage', '--verbose', '--class',
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
