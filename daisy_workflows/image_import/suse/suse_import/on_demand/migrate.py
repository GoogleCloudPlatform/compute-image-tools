#!/usr/bin/env python3
# Copyright 2020 Google Inc. All Rights Reserved.
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
"""Migrate a SLES guest to use on-demand licensing."""

import hashlib
import logging
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import threading
import typing
import urllib.request

import guestfs
from on_demand import validate_chroot
import pkg_resources

# Python dependencies that are used by the RPMs in SLES's RPM tarball,
# but not included. They are used to register the guest with SLES'S
# GCP SCC servers, and then discarded in favor of zypper packages.
#
# This is required since the guest may not have an active
# subscription, so missing dependencies cannot
# be installed from zypper.
_pip_deps = ['requests']

log = logging.getLogger('sles-on-demand')


def migrate(g: guestfs.GuestFS, tar_url: str, tar_sha256: str,
            cloud_product: str, post_convert_packages: typing.List[str]):
  """Update the guest mounted at 'g' to use GCP's on-demand subscription.

  Args:
    g: Guest to update.
    tar_url: Tarball of RPMs to bootstrap registration with.
      Expected to be from [1] or a cache of it.
    tar_sha256: SHA 256 checksum for tarball.
    cloud_product: Name of sle-module-public-cloud product, eg:
      sle-module-public-cloud/12/x86_64
    post_convert_packages: Packages to install after converting to on-demand.

  1. https://www.suse.com/support/kb/doc/?id=000019633
  """

  staging_fs_root, staged_runtime_dir = _stage_files(
      g=g,
      post_convert_packages=post_convert_packages,
      pre_convert_py=_pip_deps,
      rpm_tar_url=tar_url,
      rpm_tar_sha256=tar_sha256,
      cloud_product=cloud_product)

  chroot, guest_runtime_dir = _write_staged_to_guest(
      g, staged_runtime_dir, staging_fs_root)

  _perform_migration(g, chroot, guest_runtime_dir)

  _unmount_guest(g, chroot, guest_runtime_dir)


def _download_rpm_deps(tar_url: str, tar_sha256: str, dst: Path):
  """Downloads a tarball, verifies its checksum, and extracts its contents.

  Key points:
   - Existing permissions are replaced with:
     - owner: user ID for this process.
     - permission: read-only: 0o440
   - Files are extracted without intermediate paths

  Args:
    tar_url: URL of tarball to download.
    tar_sha256: SHA 256 checksum of tarball.
    dst: Location to extract contents of tarball.
  """
  dst.mkdir(parents=True, exist_ok=True)
  log.debug('Downloading RPM dependencies from {url}'.format(url=tar_url))
  tarpath, resp = urllib.request.urlretrieve(tar_url)
  with open(tarpath, 'rb') as f:
    actual_checksum = hashlib.sha256(f.read()).hexdigest()
  if actual_checksum != tar_sha256:
    raise AssertionError(
        'Checksum mismatch: expected: <{expected}> actual: <{actual}>'.format(
            expected=tar_sha256, actual=actual_checksum))

  tar = tarfile.open(tarpath, 'r:gz')
  for m in tar:
    if not m.isfile():
      continue
    fs_path = Path(dst, os.path.basename(m.name))
    with fs_path.open('wb') as rpm:
      rpm.write(tar.extractfile(m).read())
    fs_path.chmod(0o440)


def _download_pre_convert_py(dst: Path, pip_dependencies: typing.List[str]):
  """Installs pip dependencies to dst."""
  dst.mkdir(parents=True, exist_ok=True)
  cmd = ['pip3', 'install', '-t',
         dst.absolute().as_posix()] + pip_dependencies
  logging.info('running: {}'.format(cmd))
  subprocess.run(cmd, check=True)
  subprocess.run(['ls', '-lah', dst.absolute().as_posix()], shell=True)


def _stage_etc_hosts(g: guestfs.GuestFS, dst: Path):
  """Stage copy of guest's /etc/hosts at dst with metadata server added."""
  original = ''
  if g.exists('/etc/hosts'):
    original = g.cat('/etc/hosts')

  metadata_host_line = None
  with open('/etc/hosts') as worker_etc_hosts:
    for line in worker_etc_hosts:
      if 'metadata.google.internal' in line:
        metadata_host_line = line
        break
  if not metadata_host_line:
    raise AssertionError('Did not find metadata host in worker\'s /etc/hosts')

  dst.write_text('{}\n{}\n'.format(original, metadata_host_line))


def _remove_existing_subscriptions(g: guestfs.GuestFS):
  """Remove existing subscriptions from guest.

  Implements steps from https://www.suse.com/support/kb/doc/?id=000019085
  """
  for cmd in ['rm -f /etc/SUSEConnect',
              'rm -f /etc/zypp/{repos,services,credentials}.d/*']:
    g.sh(cmd)

  updated = []
  skip = False
  for line in g.cat('/etc/hosts').splitlines():
    if skip:
      skip = False
    elif '# Added by SMT reg' in line:
      log.info('Removing previous SMT host from /etc/hosts')
      skip = True
    else:
      updated.append(line)
  g.write('/etc/hosts', '\n'.join(updated))


def _unmount_guest(
    g: guestfs.GuestFS,
    chroot: Path,
    guest_runtime_dir: Path):
  mounts = [
      ['umount', '-l', 'proc/'],
      ['umount', '-l', 'sys/'],
      ['umount', '-l', 'dev/'],
      ['umount', '-l', guest_runtime_dir.as_posix()]
  ]
  for m in mounts:
    print('unmounting %s', m)
    subprocess.run(m, check=True, cwd=chroot.as_posix())
  g.umount_local()


def _perform_migration(
    g: guestfs.GuestFS,
    chroot: Path,
    guest_runtime_dir: Path):
  """Execute run_in_chroot.sh inside the chroot."""
  log.info('registering as on-demand')
  _remove_existing_subscriptions(g)
  run_in_chroot = (guest_runtime_dir / 'run_in_chroot.sh').relative_to(
    chroot).as_posix()
  cmd = ['/usr/sbin/chroot', chroot.as_posix(), '/bin/bash',
         run_in_chroot]
  print('chroot cmd={}'.format(cmd))
  res = subprocess.run(cmd)
  if res.returncode != 0:
    raise RuntimeError('Failed to register as on-demand.')
  log.info('registering as on-demand...done')


def _write_staged_to_guest(
    g: guestfs.GuestFS,
    staged_runtime_dir: Path,
    staging_fs_root: Path):
  """Write the staged files to the guest.

  Returns:
    Path to mounted guest, path to runtime directory inside guest
  """
  chroot = Path(tempfile.mkdtemp(prefix='guest_chroot'))
  runtime_dir = Path(chroot, g.mkdtemp('/tmp/gceXXXXXX')[1:])

  log.info('writing network files')
  g.copy_in((staging_fs_root / 'etc/hosts').as_posix(), '/etc')
  g.copy_in((staging_fs_root / 'etc/resolv.conf').as_posix(), '/etc')
  log.info('writing network files...done')

  log.info('setting up chroot')
  g.mount_local(chroot.as_posix())
  thread = threading.Thread(target=g.mount_local_run, daemon=True)
  thread.start()
  log.info('setting up chroot...done')

  log.info('setting up mounts')
  host_runtime = staged_runtime_dir.as_posix()
  chroot_runtime = runtime_dir.relative_to(chroot).as_posix()
  mounts = [
      ['mount', '-t', 'proc', '/proc', 'proc/'],
      ['mount', '--rbind', '/sys', 'sys/'],
      ['mount', '--rbind', '/dev', 'dev/'],
      ['mount', '--rbind', host_runtime, chroot_runtime]
  ]
  for cmd in mounts:
    print('mounting {}'.format(cmd))
    subprocess.run(cmd, check=True, cwd=chroot.as_posix())
  log.info('setting up mounts...done')

  _validate_directory_structure(fs_root=chroot, runtime_dir=runtime_dir,
                                check_os_mounts=True)
  return chroot, runtime_dir


def _stage_files(
    g: guestfs.GuestFS,
    post_convert_packages: typing.List[str],
    pre_convert_py: typing.List[str],
    rpm_tar_url: str,
    rpm_tar_sha256: str,
    cloud_product: str):
  """Setup files and directories that will be written to the chroot.

  Args:
    g: Mounted guest
    post_convert_packages: List of packages to install after conversion.
    pre_convert_py: List of python packages required to perform conversion.
    rpm_tar_url: URL of tarball of RPMs.
    rpm_tar_sha256: Checksum to validate tarball.
    cloud_product: Name of cloud product to install.

  Returns:
    Directory containing staged fs root
    Directory containing staged runtime root

  """
  fs_root = Path(tempfile.mkdtemp(prefix='staged_fs_root'))
  (fs_root / 'etc').mkdir(exist_ok=True, parents=True)
  runtime_dir = Path(tempfile.mkdtemp(prefix='staged_runtime_dir'))

  # cloud_product.txt
  (runtime_dir / 'cloud_product.txt').write_text(cloud_product)

  # post_convert_packages.txt
  (runtime_dir / 'post_convert_packages.txt').write_text(
      '\n'.join(post_convert_packages))

  # run_in_chroot.sh
  (runtime_dir / 'run_in_chroot.sh').write_bytes(
      pkg_resources.resource_string(__name__, 'run_in_chroot.sh'))

  # pre_convert_py
  _download_pre_convert_py(runtime_dir / 'pre_convert_py', pre_convert_py)

  # pre_convert_rpm
  _download_rpm_deps(rpm_tar_url, rpm_tar_sha256,
                     runtime_dir / 'pre_convert_rpm')

  # /etc/hosts
  _stage_etc_hosts(g, fs_root / 'etc/hosts')

  # /etc/resolv.conf
  (fs_root / 'etc/resolv.conf').write_text(
      Path('/etc/resolv.conf').read_text())

  _validate_directory_structure(fs_root, runtime_dir, check_os_mounts=False)
  return fs_root, runtime_dir


def _validate_directory_structure(fs_root: Path, runtime_dir: Path,
                                  check_os_mounts: bool):
  errs = validate_chroot.check_root(fs_root, runtime_dir,
                                    check_os_mounts=check_os_mounts)
  if errs:
    for e in errs:
      log.error(e)
    raise AssertionError('Failed to setup root, {}'.format(errs))
