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

from enum import Enum
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
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
'''

repo_sdk = '''
[google-cloud-sdk]
name=Google Cloud SDK
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el%s-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
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


class Distro(Enum):
  CENTOS = 1
  RHEL = 2


class TranslateSpec:
  def __init__(self, g: guestfs.GuestFS, use_rhel_gce_license: bool,
               distro: Distro, install_gce: bool, el_release: str):
    self.g = g
    self.use_rhel_gce_license = use_rhel_gce_license
    self.distro = distro
    self.install_gce = install_gce
    self.el_release = el_release


def check_repos(spec: TranslateSpec) -> str:
  """Check for unreachable repos.

  YUM fails if any of its repos are unreachable. Running `yum updateinfo`
  will have a non-zero return code when it fail to update any of its repos.
  """
  if run(spec.g, 'yum --help | grep updateinfo', raiseOnError=False).code != 0:
    logging.debug('command `yum updateinfo` not available. skipping test.')
    return ''
  v = 'yum updateinfo -v'
  p = run(spec.g, v, raiseOnError=False)
  logging.debug('yum updateinfo -v: {}'.format(p))
  if p.code != 0:
    return 'Ensure all configured repos are reachable.'


def yum_is_on_path(spec: TranslateSpec) -> bool:
  """Check whether the `yum` command is available."""
  p = run(spec.g, 'yum --version', raiseOnError=False)
  logging.debug('yum --version: {}'.format(p))
  return p.code == 0


def package_is_installed(spec: TranslateSpec, package: str) -> bool:
  """Check whether package is installed."""
  p = run(spec.g, ['rpm', '-q', package], raiseOnError=False)
  logging.debug('rpm -q: {}'.format(p))
  return p.code == 0


def check_yum_on_path(spec: TranslateSpec) -> str:
  """Check whether the `yum` command is available.

  If not, an error message is returned.
  """
  if not yum_is_on_path(spec):
    return 'Verify the disk\'s OS: `yum` not found.'


def check_rhel_license(spec: TranslateSpec) -> str:
  """Check for an active RHEL license.

  If a license isn't found, an error message is returned.
  """
  if spec.distro != Distro.RHEL or spec.use_rhel_gce_license:
    return ''

  p = run(spec.g, 'subscription-manager status', raiseOnError=False)
  logging.debug('subscription-manager: {}'.format(p))
  if p.code != 0:
    return 'subscription-manager did not find an active subscription. ' \
           'Omit `-byol` to register with on-demand licensing.'


def reset_network_for_dhcp(spec: TranslateSpec):
  logging.info('Resetting network to DHCP for eth0.')
  spec.g.write('/etc/sysconfig/network-scripts/ifcfg-eth0', ifcfg_eth0)
  # Remove NetworkManager-config-server if it's present. The package configures
  # NetworkManager to *not* use DHCP.
  #  https://access.redhat.com/solutions/894763
  pkg = 'NetworkManager-config-server'
  if package_is_installed(spec, pkg):
    if yum_is_on_path(spec):
      run(spec.g, ['yum', 'remove', '-y', pkg])
    else:
      run(spec.g, ['rpm', '--erase', pkg])


# If translate fails, these functions are executed in order to generate
# a more helpful error message for the user. If a check passes (returns
# a non-empty string), the search stops, and the string is shown to the user.
checks = [
    check_yum_on_path, check_repos, check_rhel_license
]


def DistroSpecific(spec: TranslateSpec):
  g = spec.g
  el_release = spec.el_release
  # This must be performed prior to making network calls from the guest.
  # Otherwise, if /etc/resolv.conf is present, and has an immutable attribute,
  # guestfs will fail with:
  #
  #   rename: /sysroot/etc/resolv.conf to
  #     /sysroot/etc/i9r7obu6: Operation not permitted
  utils.common.ClearEtcResolv(g)

  # Some imported images haven't contained `/etc/yum.repos.d`.
  if not g.exists('/etc/yum.repos.d'):
    g.mkdir('/etc/yum.repos.d')

  if spec.distro == Distro.RHEL:
    if spec.use_rhel_gce_license:
      run(g, ['yum', 'remove', '-y', '*rhui*'])
      logging.info('Adding in GCE RHUI package.')
      g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
      yum_install(g, 'google-rhui-client-rhel' + el_release)

  # Historically, translations have failed for corrupt dbcache and rpmdb.
  if yum_is_on_path(spec):
    run(g, 'yum clean -y all')

  if spec.install_gce:
    logging.info('Installing GCE packages.')
    g.write('/etc/yum.repos.d/google-cloud.repo', repo_compute % el_release)
    if el_release == '6':
      # yum operations fail when the epel repo is used with stale
      # ca-certificates, causing translation to fail. To avoid that,
      # update ca-certificates when the epel repo is found.
      #
      # The `--disablerepo` flag does the following:
      #  1. Skip the epel repo for *this* operation only.
      #  2. Block update if the epel repo isn't found.
      p = run(g, 'yum update -y ca-certificates --disablerepo=epel',
              raiseOnError=False)
      logging.debug('Attempted conditional update of '
                    'ca-certificates. Success expected only '
                    'if epel repo is installed. Result={}'.format(p))

      # Install Google Cloud SDK from the upstream tar and create links for the
      # python27 SCL environment.
      logging.info('Installing python27 from SCL.')
      yum_install(g, 'python27')
      logging.info('Installing Google Cloud SDK from tar.')
      sdk_base_url = 'https://dl.google.com/dl/cloudsdk/channels/rapid'
      sdk_base_tar = '%s/google-cloud-sdk.tar.gz' % sdk_base_url
      tar = utils.HttpGet(sdk_base_tar)
      g.write('/tmp/google-cloud-sdk.tar.gz', tar)
      run(g, ['tar', 'xzf', '/tmp/google-cloud-sdk.tar.gz', '-C', '/tmp'])
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
      run(g, ['tar', 'xzf', sdk_version_tar_file, '-C',
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
        logging.info('Enabling rsyslog')
        run(g, 'chkconfig rsyslog on')
    else:
      g.write_append(
          '/etc/yum.repos.d/google-cloud.repo', repo_sdk % el_release)
      yum_install(g, 'google-cloud-sdk')
    yum_install(g, 'google-compute-engine', 'google-osconfig-agent')

  logging.info('Updating initramfs')
  for kver in g.ls('/lib/modules'):
    logging.debug('Updating initramfs for ' + kver)
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

    # We perform a best-effort attempt to rebuild initramfs; if there's a
    # failure, continue while giving the user some debug tips. This is
    # sensible since the existing initramfs may work for booting. Or the
    # failure may be associated with an older kernel that will never be used.
    cmds = []
    if not g.exists(os.path.join('/lib/modules', kver, 'modules.dep')):
      cmds.append(['depmod', kver])
    if el_release == '6':
      # Version 6 doesn't have option --kver
      cmds.append(['dracut', '-v', '-f', kver])
    else:
      cmds.append(['dracut', '--stdlog=1', '-f', '--kver', kver])
    for cmd in cmds:
      try:
        run(g, cmd)
      except RuntimeError as e:
        cmd_string = ' '.join(cmd)
        logging.debug('`{cmd}` error: {err}'.format(cmd=cmd_string, err=e))
        msg = ('Failed to write initramfs for {kver}. If the image '
               'fails to boot: Boot the original machine, remove unused '
               'kernel versions, verify that `{cmd}` runs, re-export '
               'the disks, and re-import.').format(kver=kver, cmd=cmd_string)
        logging.info(msg)
        break

  logging.info('Update grub configuration')
  if el_release == '6':
    # Version 6 doesn't have grub2, file grub.conf needs to be updated by hand
    g.write('/tmp/grub_gce_generated', grub_cfg)
    run(g,
        r'grep -P "^[\t ]*initrd|^[\t ]*root|^[\t ]*kernel|^[\t ]*title" '
        r'/boot/grub/grub.conf >> /tmp/grub_gce_generated;'
        r'sed -i "s/console=ttyS0[^ ]*//g" /tmp/grub_gce_generated;'
        r'sed -i "/^[\t ]*kernel/s/$/ console=ttyS0,38400n8/" '
        r'/tmp/grub_gce_generated;'
        r'mv /tmp/grub_gce_generated /boot/grub/grub.conf')
  else:
    g.write('/etc/default/grub', grub2_cfg)
    run(g, ['grub2-mkconfig', '-o', '/boot/grub2/grub.cfg'])

  reset_network_for_dhcp(spec)


def yum_install(g, *packages):
  """Install one or more packages using YUM.

  Args:
    g (guestfs.GuestFS): A mounted GuestFS instance.
    *packages (list of str): The YUM packages to be installed.

  Raises:
      RuntimeError: If there is a failure during installation.
  """
  p = None
  for i in range(6):
    # There's no sleep on the first iteration since `i` is zero.
    time.sleep(i**2)
    # Bypass HTTP proxies configured in the guest image to allow
    # import to continue when the proxy is unreachable.
    #   no_proxy="*": Disables proxies set by using the `http_proxy`
    #                 environment variable.
    #   proxy=_none_: Disables proxies set in /etc/yum.conf.
    cmd = 'no_proxy="*" yum install --setopt=proxy=_none_ -y ' + ' '.join(
        '"{0}"'.format(p) for p in packages)
    p = run(g, cmd, raiseOnError=False)
    if p.code == 0:
      return
    logging.debug('Yum install failed: {}'.format(p))
  raise RuntimeError('Failed to install {}. Details: {}.'.format(
      ', '.join(packages), p))


def run_translate(g: guestfs.GuestFS):
  if (g.exists('/etc/redhat-release')
      and 'Red Hat' in g.cat('/etc/redhat-release')):
    distro = Distro.RHEL
  else:
    distro = Distro.CENTOS

  use_rhel_gce_license = utils.GetMetadataAttribute('use_rhel_gce_license')
  el_release = utils.GetMetadataAttribute('el_release')
  install_gce = utils.GetMetadataAttribute('install_gce_packages')
  spec = TranslateSpec(g=g,
                       use_rhel_gce_license=use_rhel_gce_license == 'true',
                       distro=distro,
                       el_release=el_release,
                       install_gce=install_gce == 'true')

  try:
    DistroSpecific(spec)
  except BaseException as raised:
    logging.debug('Translation failed due to: {}'.format(raised))
    for check in checks:
      msg = check(spec)
      if msg:
        raise RuntimeError('{} {}'.format(msg, raised)) from raised
    raise raised


def cleanup(g: guestfs.GuestFS):
  """Shutdown and close the guestfs.GuestFS handle, retrying on failure."""
  success = False
  for i in range(6):
    try:
      logging.debug('g.shutdown(): {}'.format(g.shutdown()))
      logging.debug('g.close(): {}'.format(g.close()))
      success = True
      break
    except BaseException as raised:
      logging.debug('cleanup failed due to: {}'.format(raised))
      logging.debug('try again in 10 seconds')
      time.sleep(10)
  if not success:
    logging.debug('Shutdown failed. Continuing anyway.')


def main():
  disk = '/dev/sdb'
  g = diskutils.MountDisk(disk)
  run_translate(g)
  utils.CommonRoutines(g)
  cleanup(g)
  utils.Execute(['virt-customize', '-a', disk, '--selinux-relabel'])


if __name__ == '__main__':
  utils.RunTranslate(main, run_with_tracing=False)
