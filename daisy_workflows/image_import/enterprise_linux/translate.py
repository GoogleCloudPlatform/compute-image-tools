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

"""Translate the EL image on a GCE VM.

Parameters (retrieved from instance metadata):

el_release: The version of the distro (6 or 7)
install_gce_packages: True if GCE agent and SDK should be installed
use_rhel_gce_license: True if GCE RHUI package should be installed
"""

import logging
import os
import re
import time

import guestfs
import utils
import utils.diskutils as diskutils
from utils.guestfsprocess import run

repo_compute = '''
[google-compute-engine]
name=Google Compute Engine
baseurl=https://packages.cloud.google.com/yum/repos/google-compute-engine-el%s-x86_64-stable
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
GRUB_CMDLINE_LINUX="crashkernel=auto console=ttyS0,38400n8 elevator=noop"
GRUB_DISABLE_RECOVERY="true"
'''

grub_cfg = '''
default=0
timeout=0
serial --unit=0 --speed=38400
terminal --timeout=0 serial console
'''


class NonFatalErrors:
  """Non-fatal errors that occur during translation.

  Not all errors are fatal; when a non-fatal error occurs, track it
  here. If translation fails, these can be used to create a customized
  error message.
  """
  def __init__(self):
    # When `yum` isn't found, either the wrong --os was specified, or
    # the disk contains EL but wasn't mountable by GuestOS.
    self.yum_command_not_found = False

    # BYOL was specified, but an active subscription wasn't found.
    self.subscription_manager_status_error = False

    # YUM will fail if any of its repositories are unreachable. Typically
    # this occurs when there's a local repo that isn't available during
    # translate. Think: DVD, intranet, NFS.
    self.yum_unreachable_repo = False

  def __repr__(self):
    return str(self.__dict__)


def check_repos(g: guestfs.GuestFS, errs: NonFatalErrors):
  """Check for unreachable repos.

  YUM fails if any of its repos are unreachable. Running `yum updateinfo`
  will have a non-zero return code when it fail to update any of its repos.
  """
  v = 'yum updateinfo -v'
  p = run(g, v)
  logging.debug('yum updateinfo -v: {}'.format(p))
  if p.code != 0:
    errs.yum_unreachable_repo = True


def check_yum_on_path(g: guestfs.GuestFS, errs: NonFatalErrors):
  """Check whether the `yum` command is available.

  If `yum` isn't found, errs is updated.
  """
  p = run(g, 'yum --version')
  logging.debug('yum --version: {}'.format(p))
  if p.code != 0:
    errs.yum_command_not_found = True


def check_rhel_license(g: guestfs.GuestFS, errs: NonFatalErrors):
  """Check for an active RHEL license.

  If a license isn't found, errs is updated.
  """
  p = run(g, 'subscription-manager status')
  logging.debug('subscription-manager: {}'.format(p))
  if p.code != 0:
    errs.subscription_manager_status_error = True


def upgrade_ca_certificates(g: guestfs.GuestFS):
  """Upgrade ca-certificates, if it is installed.

  Stale or missing CA certificates can cause yum operations to fail.
  """

  p = run(g, 'rpm -qa ca-certificates | grep ca-certificates')
  if p.code != 0:
    logging.debug(
        'ca-certificates not found. Skipping upgrade. {}'.format(p))
    return

  upgraded = False
  try:
    g.sh('yum upgrade -y ca-certificates')
    upgraded = True
  except RuntimeError as e:
    logging.debug('Failed to upgrade ca-certificates', e)
    try:
      # A common failure is older versions of EL where the user has added the
      # epel repository, and the epel repository is failing to verify.
      if 'epel' in str(e):
        g.sh('yum upgrade -y ca-certificates --disablerepo=epel')
        upgraded = True
    except RuntimeError as e:
      logging.debug('Failed second attempt to upgrade ca-certificates', e)

  if upgraded:
    logging.info('Upgraded ca-certificates package.')
  else:
    logging.info('Failed to upgrade ca-certificates package. If import '
                 'fails, update the package manually and try again.')


def DistroSpecific(g: guestfs.GuestFS, errs: NonFatalErrors):
  el_release = utils.GetMetadataAttribute('el_release')
  install_gce = utils.GetMetadataAttribute('install_gce_packages')
  rhel_license = utils.GetMetadataAttribute('use_rhel_gce_license')

  # This must be performed prior to making network calls from the guest.
  # Otherwise, if /etc/resolv.conf is present, and has an immutable attribute,
  # guestfs will fail with:
  #
  #   rename: /sysroot/etc/resolv.conf to
  #     /sysroot/etc/i9r7obu6: Operation not permitted
  utils.common.ClearEtcResolv(g)

  check_yum_on_path(g, errs)

  # Some imported images haven't contained `/etc/yum.repos.d`.
  if not g.exists('/etc/yum.repos.d'):
    g.mkdir('/etc/yum.repos.d')

  if 'Red Hat' in g.cat('/etc/redhat-release'):
    if rhel_license == 'true':
      g.command(['yum', 'remove', '-y', '*rhui*'])
      logging.info('Adding in GCE RHUI package.')
      g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
      yum_install(g, 'google-rhui-client-rhel' + el_release)
    else:
      check_rhel_license(g, errs)

  check_repos(g, errs)

  # Historically, translations have failed for corrupt dbcache and rpmdb.
  g.sh('yum clean -y all')

  upgrade_ca_certificates(g)

  if install_gce == 'true':
    logging.info('Installing GCE packages.')
    g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
    if el_release == '6':
      if 'CentOS' in g.cat('/etc/redhat-release'):
        logging.info('Installing CentOS SCL.')
        g.command(['rm', '-f', '/etc/yum.repos.d/CentOS-SCL.repo'])
        yum_install(g, 'centos-release-scl')
      # Install Google Cloud SDK from the upstream tar and create links for the
      # python27 SCL environment.
      logging.info('Installing python27 from SCL.')
      yum_install(g, 'python27')
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
      logging.info('Getting versioned Cloud SDK tar file from %s',
                   sdk_version_tar_url)
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
        bin_str = '#!/bin/bash\nsource /opt/rh/python27/enable\n%s $@' % \
            binary_path
        g.write(new_bin_path, bin_str)
        g.chmod(0o755, new_bin_path)
    else:
      g.write_append(
          '/etc/yum.repos.d/google-cloud.repo', repo_sdk % el_release)
      yum_install(g, 'google-cloud-sdk')
    yum_install(g, 'google-compute-engine', 'google-osconfig-agent')

  logging.info('Updating initramfs')
  for kver in g.ls('/lib/modules'):
    # Although each directory in /lib/modules typically corresponds to a
    # kernel version  [1], that may not always be true.
    # kernel-abi-whitelists, for example, creates extra directories in
    # /lib/modules.
    #
    # Skip building initramfs if the directory doesn't look like a
    # kernel version. Emulates the version matching from depmod [2].
    #
    # 1. https://tldp.org/LDP/Linux-Filesystem-Hierarchy/html/lib.html
    # 2. https://kernel.googlesource.com/pub/scm/linux/kernel/git/mmarek/kmod
    # /+/tip/tools/depmod.c#2537
    if not re.match(r'^\d+.\d+', kver):
      logging.debug('/lib/modules/{} doesn\'t look like a kernel directory. '
                    'Skipping creation of initramfs for it'.format(kver))
      continue
    if not g.exists(os.path.join('/lib/modules', kver, 'modules.dep')):
      try:
        g.command(['depmod', kver])
      except RuntimeError as e:
        logging.info('Failed to write initramfs for {kver}. If image fails to '
                     'boot, verify that depmod /lib/modules/{kver} runs on '
                     'the original machine'.format(kver=kver))
        logging.debug('depmod error: {}'.format(e))
        continue
    if el_release == '6':
      # Version 6 doesn't have option --kver
      g.command(['dracut', '-v', '-f', kver])
    else:
      g.command(['dracut', '--stdlog=1', '-f', '--kver', kver])

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
  # Remove NetworkManager-config-server if it's present. The package configures
  # NetworkManager to *not* use DHCP.
  #  https://access.redhat.com/solutions/894763
  g.command(['yum', 'remove', '-y', 'NetworkManager-config-server'])
  g.write('/etc/sysconfig/network-scripts/ifcfg-eth0', ifcfg_eth0)


def yum_install(g, *packages):
  """Install one or more packages using YUM.

  Args:
    g (guestfs.GuestFS): A mounted GuestFS instance.
    *packages (list of str): The YUM packages to be installed.

  Raises:
      RuntimeError: If there is a failure during installation.
  """
  last_exception = None
  for i in range(6):
    try:
      # There's no sleep on the first iteration since `i` is zero.
      time.sleep(i**2)
      # Bypass HTTP proxies configured in the guest image to allow
      # import to continue when the proxy is unreachable.
      #   no_proxy="*": Disables proxies set by using the `http_proxy`
      #                 environment variable.
      #   proxy=_none_: Disables proxies set in /etc/yum.conf.
      g.sh('no_proxy="*" yum install --setopt=proxy=_none_ -y ' + ' '.join(
          '"{0}"'.format(p) for p in packages))
      return
    except Exception as e:
      last_exception = e
      logging.debug('Failed to install {}. Details: {}.'.format(packages, e))
  raise RuntimeError('Failed to install {}. Details: {}.'.format(
      packages, last_exception))


def customize_error_message(e: BaseException,
                            errs: NonFatalErrors) -> BaseException:
  """Create an error message based on PotentialErrors."""
  logging.debug('Translation failed due to: {}'.format(e))
  logging.debug('Potential errors: {}'.format(errs))
  if errs.yum_command_not_found:
    return RuntimeError('Verify the disk\'s OS: `yum` not found. {}'.format(e))
  if errs.subscription_manager_status_error:
    return RuntimeError(
        'subscription-manager did not find an active subscription. Omit '
        '`-byol` to register with on-demand licensing. {}'.format(e))
  if errs.yum_unreachable_repo:
    return RuntimeError('Ensure that all configured repos are reachable. '
                        '{}'.format(e))
  return e


def main():
  errs = NonFatalErrors()
  disk = '/dev/sdb'
  g = diskutils.MountDisk(disk)
  try:
    DistroSpecific(g, errs)
  except BaseException as e:
    raise customize_error_message(e, errs)
  utils.CommonRoutines(g)
  diskutils.UnmountDisk(g)
  utils.Execute(['virt-customize', '-a', disk, '--selinux-relabel'])


if __name__ == '__main__':
  utils.RunTranslate(main, run_with_tracing=False)
