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
is_arm: If the image is X86 or ARM
is_byos: If the image is Bring your own Service (BYOS) or Pay as you Go (PAYG)
is_eus: If the image has Extended Update Support (EUS)
is_lvm: If the image has Logical Volume Manager (LVM) support
is_sap: If the image is RHEL for SAP
package_name: The name of the RHUI package
el_release: The EL release to build.
use_dynamic_template: Use the dynamically created templates to create images
               as part of the RHEL Build Workflow Consolidation work.
               To remove once the consolidation/refactoring is complete
el_savelogs: true to ask Anaconda to save logs (for debugging).
version_lock: The minor release version that the Image is locked
              to if EUS or SAP (ex. "9.4")
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
  use_dynamic_template = utils.GetMetadataAttribute(
    'use_dynamic_template', raise_on_not_found=False).lower()

  if use_dynamic_template == 'true':
    is_arm = utils.GetMetadataAttribute(
      'is_arm', raise_on_not_found=False).lower()
    is_byos = utils.GetMetadataAttribute(
      'rhel_byos', raise_on_not_found=False).lower()
    is_eus = utils.GetMetadataAttribute(
      'is_eus', raise_on_not_found=False).lower()
    is_lvm = utils.GetMetadataAttribute(
      'is_lvm', raise_on_not_found=False).lower()
    is_sap = utils.GetMetadataAttribute(
      'rhel_sap', raise_on_not_found=False).lower()
    rhui_package_name = utils.GetMetadataAttribute(
      'rhui_package_name', raise_on_not_found=True).lower()
    version_lock = utils.GetMetadataAttribute(
      'version_lock', raise_on_not_found=False).replace("-", ".")

    if (is_eus == 'true' or is_sap == 'true') and not version_lock:
      raise Exception(
        "invalid image build config: RHEL EUS & RHEL for "
        "SAP images must be version locked")
    if is_sap == 'true' and is_arm == 'true':
      raise Exception(
        "invalid image build config: RHEL for SAP is not supported for ARM")
    if version_lock and int(version_lock.split(".")[1]) % 2 != 0:
      raise Exception(
        "invalid image build config: RHEL EUS & RHEL for SAP are only "
        "created for even number minor releases")

  logging.info('EL Release: %s' % release)
  logging.info('Build working directory: %s' % os.getcwd())

  iso_file = '/files/installer.iso'
  ks_cfg = '/files/ks.cfg'
  kickstart_vars_file = '/files/kickstart_vars.cfg'

  utils.AptGetInstall(['rsync'])

  # Write the installer disk. Write GPT label, create partition,
  # copy installer boot files over.
  logging.info('Writing installer disk.')

  installer_disk = ('/dev/' + os.path.basename(
      os.readlink('/dev/disk/by-id/google-disk-installer')))

  utils.Execute(['parted', installer_disk, 'mklabel', 'gpt'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'mkpart', 'primary', 'fat32', '1MB',
                 '2048MB'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'mkpart', 'primary', 'ext2',
                 '2048MB', '100%'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'set', '1', 'boot', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['parted', installer_disk, 'set', '1', 'esp', 'on'])
  utils.Execute(['sync'])

  installer_disk1 = ('/dev/' + os.path.basename(
      os.readlink('/dev/disk/by-id/google-disk-installer-part1')))
  installer_disk2 = ('/dev/' + os.path.basename(
      os.readlink('/dev/disk/by-id/google-disk-installer-part2')))

  utils.Execute(['mkfs.vfat', '-F', '32', installer_disk1])
  utils.Execute(['sync'])
  utils.Execute(['fatlabel', installer_disk1, 'ESP'])
  utils.Execute(['sync'])
  utils.Execute(['mkfs.ext4', '-L', 'INSTALLER', installer_disk2])
  utils.Execute(['sync'])

  utils.Execute(['mkdir', '-vp', 'iso', 'installer', 'boot'])
  utils.Execute(['mount', '-o', 'ro,loop', '-t', 'iso9660', iso_file, 'iso'])
  utils.Execute(['mount', '-t', 'vfat', installer_disk1, 'boot'])
  utils.Execute(['mount', '-t', 'ext4', installer_disk2, 'installer'])

  if use_dynamic_template == 'true':
    logging.info('Writing Kickstart variables file to installer disk.')
    with open(kickstart_vars_file, 'w') as f:
      f.write(f'IS_ARM={is_arm}\n')
      f.write(f'IS_BYOS={is_byos}\n')
      f.write(f'IS_EUS={is_eus}\n')
      f.write(f'IS_LVM={is_lvm}\n')
      f.write(f'IS_SAP={is_sap}\n')
      f.write(f'RHUI_PACKAGE_NAME={str(rhui_package_name).lower()}\n')
      if version_lock:
        f.write(f'VERSION_LOCK="{version_lock}"\n')
    logging.info(f'Successfully wrote {kickstart_vars_file}')

  utils.Execute(['cp', '-r', 'iso/EFI', 'boot/'])
  utils.Execute(['cp', '-r', 'iso/images', 'boot/'])
  utils.Execute(['cp', iso_file, 'installer/'])
  utils.Execute(['cp', ks_cfg, 'installer/'])
  if use_dynamic_template == 'true':
    utils.Execute(['cp', kickstart_vars_file, 'installer/'])

  # The kickstart config contains a preinstall script copying, reloading, and
  # triggering this rule in the install environment. This allows us to use
  # predictable names for block devices. It would be perferable to take a
  # simpler approach such as selecting the disk with an unkown partition table
  # but kickstart does not believe the default google nvme device names are
  # are deterministic and refuses to use them without user input.

  utils.Execute(['cp', '-L', '/usr/lib/udev/rules.d/65-gce-disk-naming.rules',
  'installer/'])
  utils.Execute(['cp', '-L', '/usr/lib/udev/google_nvme_id', 'installer/'])

  # Modify boot config.
  with open('boot/EFI/BOOT/grub.cfg', 'r+') as f:
    oldcfg = f.read()
    cfg = re.sub(r'-l .RHEL.*', r"""-l 'ESP'""", oldcfg)
    cfg = re.sub(r'timeout=60', 'timeout=1', cfg)
    cfg = re.sub(r'set default=.*', 'set default="0"', cfg)
    cfg = re.sub(r'load_video\n',
           r'serial --speed=115200 --unit=0 --word=8 --parity=no\n'
           'terminal_input serial\nterminal_output serial\n', cfg)

    # Change boot args.
    args = ' '.join([
      'inst.text', 'inst.ks=hd:LABEL=INSTALLER:/%s' % ks_cfg,
      'console=ttyS0,115200', 'inst.gpt', 'inst.loglevel=debug'
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

  # Update google_nvme_id to remove xxd dependency, if necessary
  # The current worker uses an older version
  with open('installer/google_nvme_id', 'r+') as f:
    old = f.read()
    new = re.sub(r'xxd -p -seek 384 \| xxd -p -r',
    'dd bs=1 skip=384 2>/dev/null',
    old)
    f.seek(0)
    f.write(new)
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
