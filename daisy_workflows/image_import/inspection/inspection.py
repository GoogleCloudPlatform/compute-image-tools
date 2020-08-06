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
"""Finds boot-related properties of disks.

This module supports offline inspection of block devices and
virtual disk files, with a focus on information that typically
requires booting the system.

See `model.InspectionResults` for which information is returned.

In terms of OS support, this module focuses on systems
that are runnable on Google Compute Engine, with a particular focus on the
systems that are importable to Google Compute Engine:
  https://cloud.google.com/compute/docs/import

In other words, it doesn't seek to exhaustively detect all systems,
and will remove support for defunct systems over time.
"""

import re
import sys

import guestfs
from inspectors.os import architecture
from inspectors.os import linux
from inspectors.os import windows
import model
import system.filesystems

_LINUX = [
  linux.Fingerprint(model.Distro.AMAZON, aliases=['amzn', 'amazonlinux']),
  linux.Fingerprint(
    model.Distro.CENTOS,
    legacy=linux.LegacyFingerprint(
      metadata_file='/etc/centos-release',
      version_pattern=re.compile(r'\d+\.\d+'),
      derivative_metadata_files=[
        '/etc/fedora-release',
        '/etc/oracle-release',
      ]),
  ),
  linux.Fingerprint(
    model.Distro.DEBIAN,
    legacy=linux.LegacyFingerprint(
      metadata_file='/etc/debian_version',
      version_pattern=re.compile(r'\d+\.\d+'),
    ),
  ),
  linux.Fingerprint(model.Distro.FEDORA),
  linux.Fingerprint(model.Distro.KALI),
  linux.Fingerprint(
    model.Distro.RHEL,
    legacy=linux.LegacyFingerprint(
      metadata_file='/etc/redhat-release',
      version_pattern=re.compile(r'\d+\.\d+'),
      derivative_metadata_files=[
        '/etc/centos-release',
        '/etc/fedora-release',
        '/etc/oracle-release',
      ]),
  ),
  linux.Fingerprint(model.Distro.SLES, aliases=['sles_sap']),
  linux.Fingerprint(model.Distro.OPENSUSE, aliases=['opensuse-leap']),
  linux.Fingerprint(model.Distro.ORACLE, aliases=['ol', 'oraclelinux']),
  linux.Fingerprint(model.Distro.UBUNTU),
]


def inspect_device(device: str) -> model.InspectionResults:
  """Finds boot-related properties for a device using offline inspection.

  Args:
    device: a reference to a mounted block device (eg: /dev/sdb), or
    to a virtual disk file (eg: /opt/images/disk.vmdk).
  """
  g = guestfs.GuestFS(python_return_dict=True)
  g.add_drive_opts(device, readonly=1)
  g.launch()

  roots = g.inspect_os()
  if len(roots) == 0:
    print('inspect_vm: no operating systems found', file=sys.stderr)
    sys.exit(1)
  root = roots[0]
  mount_points = g.inspect_get_mountpoints(root)
  for dev, mp in sorted(mount_points.items(), key=lambda k: len(k[0])):
    try:
      g.mount_ro(mp, dev)
    except RuntimeError as msg:
      print('%s (ignored)' % msg, file=sys.stderr)
  fs = system.filesystems.GuestFSFilesystem(g)
  operating_system = linux.Inspector(fs, _LINUX).inspect()
  if not operating_system:
    operating_system = windows.Inspector(g, root).inspect()
  arch = architecture.Inspector(g, root).inspect()
  g.umount_all()

  return model.InspectionResults(
    device=device,
    os=operating_system,
    architecture=arch,
  )


def _linux_inspector(fs: system.filesystems.Filesystem) -> linux.Inspector:
  """Returns a linux.Inspector that is configured

  with all detectable Linux distros.
  """
  return linux.Inspector(fs, _LINUX)
