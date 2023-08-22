#!/usr/bin/env python3
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

"""Convert EL ISO to GCE Image and prep for installation.

Parameters (retrieved from instance metadata):
el_release: The EL release to build.
el_savelogs: true to ask Anaconda to save logs (for debugging).
"""

import difflib
import logging
import os
import re

import utils


def main():
  # Get Parameters
  release = utils.GetMetadataAttribute('el_release', raise_on_not_found=True)
  savelogs = utils.GetMetadataAttribute('el_savelogs') == 'true'
  install_disk = 'scsi-0Google_PersistentDisk_'
  install_disk += utils.GetMetadataAttribute('install_disk',
  raise_on_not_found=True)

  logging.info('EL Release: %s' % release)
  logging.info('Build working directory: %s' % os.getcwd())

  iso_file = '/files/installer.iso'
  ks_cfg = '/files/ks.cfg'

  utils.AptGetInstall(['rsync'])

  # Write the installer disk. Write GPT label, create partition,
  # copy installer boot files over.
  logging.info('Writing installer disk.')

  installer_disk = ('/dev/' + os.path.basename(
      os.readlink('/dev/disk/by-id/google-disk-installer')))
  installer_disk1 = installer_disk + '1'
  installer_disk2 = installer_disk + '2'

  utils.Execute(['parted', installer_disk, 'mklabel', 'gpt'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'mkpart', 'primary', 'fat32', '1MB',
                 '1024MB'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'mkpart', 'primary', 'ext2',
                 '1024MB', '100%'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'set', '1', 'boot', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'set', '1', 'esp', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['mkfs.vfat', '-F', '32', installer_disk1])
  utils.Execute(['sync'])
  utils.Execute(['fatlabel', installer_disk1, 'ESP'])
  utils.Execute(['sync'])
  utils.Execute(['mkfs.ext2', '-L', 'INSTALLER', installer_disk2])
  utils.Execute(['sync'])

  utils.Execute(['mkdir', '-vp', 'iso', 'installer', 'boot'])
  utils.Execute(['mount', '-o', 'ro,loop', '-t', 'iso9660', iso_file, 'iso'])
  utils.Execute(['mount', '-t', 'vfat', installer_disk1, 'boot'])
  utils.Execute(['mount', '-t', 'ext2', installer_disk2, 'installer'])
  utils.Execute(['cp', '-r', 'iso/EFI', 'boot/'])
  utils.Execute(['cp', '-r', 'iso/images', 'boot/'])
  utils.Execute(['cp', iso_file, 'installer/'])
  utils.Execute(['cp', ks_cfg, 'installer/'])

  # Modify boot config.
  with open('boot/EFI/BOOT/grub.cfg', 'r+') as f:
    oldcfg = f.read()
    cfg = re.sub(r'-l .RHEL.*', r"""-l 'ESP'""", oldcfg)
    cfg = re.sub(r'timeout=60', 'timeout=1', cfg)
    cfg = re.sub(r'set default=.*', 'set default="0"', cfg)
    cfg = re.sub(r'load_video\n',
           r'serial --speed=38400 --unit=0 --word=8 --parity=no\n'
           'terminal_input serial\nterminal_output serial\n', cfg)

    # Change boot args.
    args = ' '.join([
      'inst.text', 'inst.ks=hd:LABEL=INSTALLER:/%s' % ks_cfg,
      'console=ttyS0,38400n8', 'inst.gpt', 'inst.loglevel=debug'
    ])

    # Tell Anaconda not to store its logs in the installed image,
    # unless requested to keep them for debugging.
    if not savelogs:
      args += ' inst.nosave=all'
    cfg = re.sub(r'inst\.stage2.*', r'\g<0> %s' % args, cfg)

    # Change labels to explicit partitions.
    cfg = re.sub(r'LABEL=[^ ]+', 'LABEL=INSTALLER', cfg)

    # Print out a the modifications.
    diff = difflib.Differ().compare(
        oldcfg.splitlines(1),
        cfg.splitlines(1))
    logging.info('Modified grub.cfg:\n%s' % '\n'.join(diff))

    f.seek(0)
    f.write(cfg)
    f.truncate()

  # Modify kickstart config
  with open('installer/ks.cfg', 'r+') as f:
    oldcfg = f.read()
    cfg = re.sub(r'sub-install-disk-id', install_disk, oldcfg)

    f.seek(0)
    f.write(cfg)
    f.truncate()

  utils.Execute(['umount', 'installer'])
  utils.Execute(['umount', 'iso'])
  utils.Execute(['umount', 'boot'])


if __name__ == '__main__':
  try:
    main()
    logging.success('EL Installer build successful!')
  except Exception as e:
    logging.error('EL Installer build failed: %s' % str(e))
