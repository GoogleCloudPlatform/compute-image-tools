#!/usr/bin/env python
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

"""Translate the EL image on a GCE VM.

Parameters (retrieved from instance metadata):

el_release: The version of the distro (6 or 7)
install_gce_packages: True if GCE agent and SDK should be installed
use_rhel_gce_license: True if GCE RHUI package should be installed
"""

import logging
import os

import utils

utils.AptGetInstall(['python-guestfs', 'libguestfs-tools'])

import guestfs


repo_compute = '''
[google-cloud-compute]
name=Google Cloud Compute
baseurl=https://packages.cloud.google.com/yum/repos/google-cloud-compute-el%s-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
'''

repo_sdk = '''
[google-cloud-sdk]
name=Google Cloud SDK
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el%s-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
'''

ifcfg_eth0 = '''
BOOTPROTO=dhcp
DEVICE=eth0
ONBOOT=yes
TYPE=Ethernet
DEFROUTE=yes
PEERDNS=yes
PEERROUTES=yes
DHCP_HOSTNAME=localhost
IPV4_FAILURE_FATAL=no
NAME="System eth0"
MTU=1460
PERSISTENT_DHCLIENT="y"
'''

grub2_cfg = '''
GRUB_TIMEOUT=0
GRUB_DISTRIBUTOR="$(sed 's, release .*$,,g' /etc/system-release)"
GRUB_DEFAULT=saved
GRUB_DISABLE_SUBMENU=true
GRUB_TERMINAL="serial console"
GRUB_SERIAL_COMMAND="serial --speed=38400"
GRUB_CMDLINE_LINUX="crashkernel=auto console=ttyS0,38400n8"
GRUB_DISABLE_RECOVERY="true"
'''

grub_cfg = '''
default=0
timeout=0
serial --unit=0 --speed=38400
terminal --timeout=0 serial console
'''


def DistroSpecific(g):
  el_release = utils.GetMetadataParam('el_release')
  install_gce = utils.GetMetadataParam('install_gce_packages')
  rhel_license = utils.GetMetadataParam('use_rhel_gce_license')

  if rhel_license == 'true':
    if 'Red Hat' in g.cat('/etc/redhat-release'):
      g.command(['yum', 'remove', '-y', '*rhui*'])
      logging.info('Adding in GCE RHUI package.')
      g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
      g.command(
          ['yum', 'install', '-y', 'google-rhui-client-rhel%s' % el_release])

  if install_gce == 'true':
    logging.info('Installing GCE packages.')
    g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
    if el_release == '7':
      g.write_append(
          '/etc/yum.repos.d/google-cloud.repo', repo_sdk % el_release)
      g.command(['yum', '-y', 'install', 'google-cloud-sdk'])
    if el_release == '6':
      if 'CentOS' in g.cat('/etc/redhat-release'):
        logging.info('Installing CentOS SCL.')
        g.command(['rm', '-f', '/etc/yum.repos.d/CentOS-SCL.repo'])
        g.command(['yum', '-y', 'install', 'centos-release-scl'])
      # Install Google Cloud SDK from the upstream tar and create links for the
      # python27 SCL environment.
      logging.info('Installing python27 from SCL.')
      g.command(['yum', '-y', 'install', 'python27'])
      g.command(['scl', 'enable', 'python27',
                 'pip2.7 install --upgrade google_compute_engine'])

      logging.info('Installing Google Cloud SDK from tar.')
      sdk_base_url = 'https://dl.google.com/dl/cloudsdk/channels/rapid'
      sdk_base_tar = '%s/google-cloud-sdk.tar.gz' % sdk_base_url
      tar = utils.HttpGet(sdk_base_tar)
      g.write('/tmp/google-cloud-sdk.tar.gz', tar)
      g.command(['tar', 'xzf', '/tmp/google-cloud-sdk.tar.gz', '-C', '/tmp'])
      sdk_version = g.cat('/tmp/google-cloud-sdk/VERSION').strip()

      logging.info('Getting Cloud SDK Version %s', sdk_version)
      sdk_version_tar = 'google-cloud-sdk-%s-linux-x86_64.tar.gz' % sdk_version
      sdk_version_tar_url = '%s/downloads/%s' % (sdk_base_url, sdk_version_tar)
      logging.info('Getting versioned Cloud SDK tar file from %s', sdk_version_tar_url)
      tar = utils.HttpGet(sdk_version_tar_url)
      sdk_version_tar_file = os.path.join('/tmp', sdk_version_tar)
      g.write(sdk_version_tar_file, tar)
      g.mkdir_p('/usr/local/share/google')
      g.command(['tar', 'xzf', sdk_version_tar_file, '-C',
                 '/usr/local/share/google', '--no-same-owner'])

      logging.info('Creating CloudSDK SCL symlinks.')
      sdk_bin_path = '/usr/local/share/google/google-cloud-sdk/bin'
      g.ln_s(os.path.join(sdk_bin_path, 'git-credential-gcloud.sh'),
             os.path.join('/usr/bin', 'git-credential-gcloud.sh'))
      for binary in ['bq', 'gcloud', 'gsutil']:
        binary_path = os.path.join(sdk_bin_path, binary)
        new_bin_path = os.path.join('/usr/bin', binary)
        bin_str = '#!/bin/bash\nsource /opt/rh/python27/enable\n%s $@' % binary_path
        g.write(new_bin_path, bin_str)
        g.chmod(0755, new_bin_path)

    g.command([
        'yum', '-y', 'install', 'google-compute-engine',
        'python-google-compute-engine'])

  logging.info('Updating initramfs')
  for kver in g.ls('/lib/modules'):
    if el_release == '6':
      # Version 6 doesn't have option --kver
      g.command(['dracut', '-v', '-f', kver])
    else:
      g.command(['dracut', '-v', '-f', '--kver', kver])

  logging.info('Update grub configuration')
  if el_release == '6':
    # Version 6 doesn't have grub2, file grub.conf needs to be updated by hand
    g.write('/tmp/grub_gce_generated', grub_cfg)
    g.sh(
        r'grep -P "^[\t ]*initrd|^[\t ]*root|^[\t ]*kernel|^[\t ]*title" '
            r'/boot/grub/grub.conf >> /tmp/grub_gce_generated;'
        r'sed -i "s/console=ttyS0[^ ]*//g" /tmp/grub_gce_generated;'
        r'sed -i "/^[\t ]*kernel/s/$/ console=ttyS0,38400n8/" '
            r'/tmp/grub_gce_generated;'
        r'mv /tmp/grub_gce_generated /boot/grub/grub.conf')
  else:
    g.write('/etc/default/grub', grub2_cfg)
    g.command(['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])

  # Reset network for DHCP.
  logging.info('Resetting network to DHCP for eth0.')
  g.write('/etc/sysconfig/network-scripts/ifcfg-eth0', ifcfg_eth0)


def main():
  disk = '/dev/sdb'
  g = utils.MountDisk(disk)
  DistroSpecific(g)
  utils.CommonRoutines(g)
  utils.UnmountDisk(g)
  utils.Execute(['virt-customize', '-a', disk, '--selinux-relabel'])

if __name__=='__main__':
  utils.RunTranslate(main) 
