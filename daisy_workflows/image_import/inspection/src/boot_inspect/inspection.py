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

import os
import re
import sys

from boot_inspect.inspectors.os import architecture, linux, windows
import boot_inspect.system.filesystems
from compute_image_tools_proto import inspect_pb2

_LINUX = [
    linux.Fingerprint(inspect_pb2.Distro.AMAZON,
                      aliases=['amzn', 'amazonlinux']),
    linux.Fingerprint(
        inspect_pb2.Distro.CENTOS,
        fs_predicate=linux.FileExistenceMatcher(
            require={'/etc/centos-release'},
            disallow={'/etc/fedora-release',
                      '/etc/oracle-release'}),
        version_reader=linux.VersionReader(
            metadata_file='/etc/centos-release',
            version_pattern=re.compile(r'\d+\.\d+')),
    ),
    linux.Fingerprint(
        inspect_pb2.Distro.DEBIAN,
        version_reader=linux.VersionReader(
            metadata_file='/etc/debian_version',
            version_pattern=re.compile(r'\d+\.\d+'),
        ),
    ),
    linux.Fingerprint(inspect_pb2.Distro.FEDORA),
    linux.Fingerprint(inspect_pb2.Distro.KALI),
    linux.Fingerprint(
        inspect_pb2.Distro.RHEL,
        fs_predicate=linux.FileExistenceMatcher(
            require={'/etc/redhat-release'},
            disallow={'/etc/fedora-release',
                      '/etc/oracle-release',
                      '/etc/centos-release'}),
        version_reader=linux.VersionReader(
            metadata_file='/etc/redhat-release',
            version_pattern=re.compile(r'\d+\.\d+')),
    ),
    # Depending on the version, SLES for SAP has a variety of identifiers in
    # /etc/os-release.  To match, one of those identifiers must be seen
    # *and* the file /etc/products.d/SLES_SAP.prod must exist.
    #
    # This is documented here:
    #   https://www.suse.com/support/kb/doc/?id=000019341
    linux.Fingerprint(
        inspect_pb2.Distro.SLES_SAP,
        aliases=['sles', 'sles_sap'],
        fs_predicate=linux.FileExistenceMatcher(
            require={'/etc/products.d/SLES_SAP.prod'})
    ),
    linux.Fingerprint(inspect_pb2.Distro.SLES),
    linux.Fingerprint(inspect_pb2.Distro.OPENSUSE, aliases=['opensuse-leap']),
    linux.Fingerprint(inspect_pb2.Distro.ORACLE,
                      aliases=['ol', 'oraclelinux']),
    linux.Fingerprint(inspect_pb2.Distro.UBUNTU),
]


def inspect_device(g) -> inspect_pb2.InspectionResults:
  """Finds boot-related properties for a device using offline inspection.

  Args:
    g (guestfs.GuestFS): A launched, but unmounted, GuestFS instance.

  Example:

    g = guestfs.GuestFS(python_return_dict=True)
    g.add_drive_opts("/dev/sdb", format="raw")
    g.launch()
    results = inspect_device(g, "/dev/sdb")
  """

  roots = g.inspect_os()
  if len(roots) == 0:
    return inspect_pb2.InspectionResults(
        os_count=len(roots)

    )
  root = roots[0]
  mount_points = g.inspect_get_mountpoints(root)
  for dev, mp in sorted(mount_points.items(), key=lambda k: len(k[0])):
    try:
      g.mount_ro(mp, dev)
    except RuntimeError as msg:
      print('%s (ignored)' % msg, file=sys.stderr)
  fs = boot_inspect.system.filesystems.GuestFSFilesystem(g)
  operating_system = linux.Inspector(fs, _LINUX).inspect()
  if not operating_system:
    operating_system = windows.Inspector(g, root).inspect()
  if operating_system:
    operating_system.architecture = architecture.Inspector(g, root).inspect()

  g.umount_all()

  return inspect_pb2.InspectionResults(
      os_release=operating_system,
      os_count=1,
  )


def inspect_boot_loader(g, device) -> inspect_pb2.InspectionResults:
  """Finds boot-loader properties for the device using offline inspection.

  Args:
    g (guestfs.GuestFS): A launched, but unmounted, GuestFS instance.
    device: a reference to a mounted block device (eg: /dev/sdb), or
    to a virtual disk file (eg: /opt/images/disk.vmdk).

  Example:

    g = guestfs.GuestFS(python_return_dict=True)
    g.add_drive_opts("/dev/sdb", format="raw")
    g.launch()
    results = inspect_boot_loader(g)
  """

  bios_bootable = False
  uefi_bootable = False
  root_fs = ""

  try:
    stream = os.popen('gdisk -l {}'.format(device))
    output = stream.read()
    print(output)
    if _inspect_for_hybrid_mbr(output):
      bios_bootable = True

    part_list = g.part_list('/dev/sda')
    for part in part_list:
      try:
        guid = g.part_get_gpt_type('/dev/sda', part['part_num'])
        # It covers both GPT "EFI System" and BIOS "EFI (FAT-12/16/32)".
        if guid == 'C12A7328-F81F-11D2-BA4B-00A0C93EC93B':
          uefi_bootable = True
          # TODO: detect root_fs (b/169245755)
        # It covers "BIOS boot", which make a protective-MBR bios-bootable.
        if guid == '21686148-6449-6E6F-744E-656564454649':
          bios_bootable = True
      except Exception:
        continue

  except Exception as e:
    print("Failed to inspect disk partition: ", e)

  return inspect_pb2.InspectionResults(
      bios_bootable=bios_bootable,
      uefi_bootable=uefi_bootable,
      root_fs=root_fs,
  )


def _inspect_for_hybrid_mbr(gdisk_output) -> bool:
  """Finds hybrid MBR, which potentially is BIOS bootableeven without a BIOS
   boot partition.

   Args:
     gdisk_output: output from gdisk that contains partition info.
   """
  is_hybrid_mbr = False
  mbr_bios_bootable_re = re.compile(r'(.*)MBR:[\s]*hybrid(.*)', re.DOTALL)
  if mbr_bios_bootable_re.match(gdisk_output):
    is_hybrid_mbr = True
  return is_hybrid_mbr


def _linux_inspector(
    fs: boot_inspect.system.filesystems.Filesystem) -> linux.Inspector:
  """Returns a linux.Inspector that is configured

  with all detectable Linux distros.
  """
  return linux.Inspector(fs, _LINUX)
