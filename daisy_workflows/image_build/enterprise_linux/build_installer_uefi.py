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
google_cloud_repo: The package repo to use. Can be stable (default), staging,
  or unstable.
el_release: rhel6, rhel7, centos6, centos7, oraclelinux6, or oraclelinux7
el_savelogs: true to ask Anaconda to save logs (for debugging).
rhel_byol: true if building a RHEL BYOL image.
"""

import difflib
import logging
import os
import re

import ks_helpers
import utils


def main():
  # Get Parameters
  repo = utils.GetMetadataAttribute('google_cloud_repo',
                    raise_on_not_found=True)
  release = utils.GetMetadataAttribute('el_release', raise_on_not_found=True)
  savelogs = utils.GetMetadataAttribute('el_savelogs') == 'true'
  byol = utils.GetMetadataAttribute('rhel_byol') == 'true'
  sap = utils.GetMetadataAttribute('rhel_sap') == 'true'
  uefi = utils.GetMetadataAttribute('rhel_uefi') == 'true'
  nge = utils.GetMetadataAttribute('new_guest',
                                   raise_on_not_found=False) == 'true'

  logging.info('EL Release: %s' % release)
  logging.info('Google Cloud repo: %s' % repo)
  logging.info('Build working directory: %s' % os.getcwd())

  iso_file = '/files/installer.iso'

  # Necessary libs and tools to build the installer disk.
  utils.AptGetInstall(['dosfstools', 'rsync'])

  # Build the kickstart file.
  ks_content = ks_helpers.BuildKsConfig(release, repo, byol, sap, uefi, nge)
  ks_cfg = 'ks.cfg'
  utils.WriteFile(ks_cfg, ks_content)

  # Write the installer disk. Write GPT label, create partition,
  # copy installer boot files over.
  logging.info('Writing installer disk.')
  utils.Execute(['parted', '/dev/sdb', 'mklabel', 'gpt'])
  utils.Execute(['sync'])
  utils.Execute(['parted', '/dev/sdb', 'mkpart', 'primary', 'fat32', '1MB',
                 '601MB'])
  utils.Execute(['sync'])
  utils.Execute(['parted', '/dev/sdb', 'mkpart', 'primary', 'ext2', '601MB',
                 '100%'])
  utils.Execute(['sync'])
  utils.Execute(['parted', '/dev/sdb', 'set', '1', 'boot', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['parted', '/dev/sdb', 'set', '1', 'esp', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['mkfs.vfat', '-F', '32', '/dev/sdb1'])
  utils.Execute(['sync'])
  utils.Execute(['fatlabel', '/dev/sdb1', 'ESP'])
  utils.Execute(['sync'])
  utils.Execute(['mkfs.ext2', '-L', 'INSTALLER', '/dev/sdb2'])
  utils.Execute(['sync'])

  utils.Execute(['mkdir', '-vp', 'iso', 'installer', 'boot'])
  utils.Execute(['mount', '-o', 'ro,loop', '-t', 'iso9660', iso_file, 'iso'])
  utils.Execute(['mount', '-t', 'vfat', '/dev/sdb1', 'boot'])
  utils.Execute(['mount', '-t', 'ext2', '/dev/sdb2', 'installer'])
  utils.Execute(['rsync', '-Pav', 'iso/EFI', 'iso/images', 'boot/'])
  utils.Execute(['cp', iso_file, 'installer/'])
  utils.Execute(['cp', ks_cfg, 'installer/'])
  utils.Execute(['cp', '-r', '/files/sb_keys', 'installer/'])

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
      'text', 'ks=hd:LABEL=INSTALLER:/%s' % ks_cfg,
      'console=ttyS0,38400n8', 'inst.gpt', 'loglevel=debug'
    ])

    # Tell Anaconda not to store its logs in the installed image,
    # unless requested to keep them for debugging.
    if not savelogs:
      args += ' inst.nosave=all'
    cfg = re.sub(r'inst\.stage2.*', r'\g<0> %s' % args, cfg)

    if release in ['centos7', 'rhel7', 'oraclelinux7', 'centos8', 'rhel8']:
      cfg = re.sub(r'LABEL=[^ :]+', 'LABEL=INSTALLER', cfg)

    # Print out a the modifications.
    diff = difflib.Differ().compare(
        oldcfg.splitlines(1),
        cfg.splitlines(1))
    logging.info('Modified grub.cfg:\n%s' % '\n'.join(diff))

    f.seek(0)
    f.write(cfg)
    f.truncate()

  logging.info("Creating gsetup boot path file\n")
  utils.Execute(['mkdir', '-p', 'boot/EFI/Google/gsetup'])
  with open('boot/EFI/Google/gsetup/boot', 'w') as g:
    g.write("\\EFI\\BOOT\\grubx64.efi\n")

  utils.Execute(['umount', 'installer'])
  utils.Execute(['umount', 'iso'])
  utils.Execute(['umount', 'boot'])


if __name__ == '__main__':
  try:
    main()
    logging.success('EL Installer build successful!')
  except Exception as e:
    logging.error('EL Installer build failed: %s' % str(e))
