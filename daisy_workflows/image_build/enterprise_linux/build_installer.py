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

"""Convert EL ISO to GCE Image and prep for installation.

Parameters (retrieved from instance metadata):
google_cloud_repo: The package repo to use. Can be stable (default), staging, or unstable.
el_release: rhel6, rhel7, centos6, or centos7
"""
import difflib
import logging
import os
import re

import ks_helpers
import utils


def main():
  # Get Parameters
  repo = utils.GetMetadataParam('google_cloud_repo', raise_on_not_found=True)
  release = utils.GetMetadataParam('el_release', raise_on_not_found=True)

  logging.info('EL Installer Builder')
  logging.info('==============')
  logging.info('Release: %s', release)
  logging.info('Google Cloud repo: %s', repo)
  logging.info('Build working directory: %s', os.getcwd())

  iso_file = 'installer.iso'

  # Necessary libs and tools to build the installer disk.
  utils.AptGetInstall(['extlinux', 'rsync'])

  # Build the kickstart file.
  ks_content = ks_helpers.BuildKsConfig(release, repo)
  ks_cfg = 'ks.cfg'
  utils.WriteFile(ks_cfg, ks_content)

  # Write the installer disk. Write extlinux MBR, create partition,
  # copy installer ISO and ISO boot files over.
  logging.info('Writing installer disk.')
  utils.Execute(['parted', '/dev/sdb', 'mklabel', 'msdos'])
  utils.Execute(['parted', '/dev/sdb', 'mkpart', 'primary', '1MB', '100%'])
  utils.Execute(['parted', '/dev/sdb', 'set', '1', 'boot', 'on'])
  utils.Execute(['sync'])
  utils.Execute(['dd', 'if=/usr/lib/EXTLINUX/mbr.bin', 'of=/dev/sdb'])
  utils.Execute(['mkfs.ext4', '/dev/sdb1'])
  utils.Execute(['mkdir', 'iso', 'installer'])
  utils.Execute(['mount', '-o', 'ro,loop', '-t', 'iso9660', iso_file, 'iso'])
  utils.Execute(['mount', '-t', 'ext4', '/dev/sdb1', 'installer'])
  utils.Execute(['rsync', '-Pav', 'iso/images', 'iso/isolinux', 'installer/'])
  utils.Execute(['cp', iso_file, 'installer/'])
  utils.Execute(['cp', ks_cfg, 'installer/'])
  if release in ['rhel6', 'rhel7']:
    logging.info('Copying RHUI client RPM from %s', rhui_client_rpm)
    rhui_client_rpm = utils.GetMetadataParam('rhui_client_rpm',
                                             raise_on_not_found=True)
    utils.Execute(['gsutil', 'cp', rhui_client_rpm, 'installer/'])

  # Modify boot files on installer disk.
  utils.Execute(['mv', 'installer/isolinux', 'installer/extlinux'])
  utils.Execute(
      ['mv', 'installer/extlinux/isolinux.cfg',
       'installer/extlinux/extlinux.conf'])

  # Modify boot config.
  with open('installer/extlinux/extlinux.conf', 'r+') as f:
    oldcfg = f.read()
    cfg = re.sub(r'^default.*', r'default linux', oldcfg, count=1)

    # Change boot args.
    args = ' '.join([
        'text', 'ks=hd:/dev/sda1:/%s' % ks_cfg,
        'console=ttyS0,38400n8', 'sshd=1', 'loglevel=debug'
    ])
    cfg = re.sub(r'append initrd=initrd\.img.*', r'\g<0> %s' % args, cfg)

    # Change labels to explicit partitions.
    # Note that RHEL ISO's are keyed off of the release ID, RHEL-7.3 for
    # example. The following command will give you the string:
    # isoinfo -d -i rhel-server.iso | grep "Volume id:"
    if release == 'rhel7':
      cfg = re.sub(r'LABEL=RHEL-7.3\\x20Server\.x86_64', '/dev/sda1', cfg)
    elif release == 'centos7':
      cfg = re.sub(r'LABEL=CentOS\\x207\\x20x86_64', '/dev/sda1', cfg)

    # Print out a the modifications.
    diff = difflib.Differ().compare(oldcfg.splitlines(1), cfg.splitlines(1))
    logging.info('Modified extlinux.conf:\n%s', '\n'.join(diff))

    f.seek(0)
    f.write(cfg)
    f.truncate()

  # Activate extlinux.
  utils.Execute(['extlinux', '--install', 'installer/extlinux'])


if __name__ == '__main__':
  utils.RunScript(main)
