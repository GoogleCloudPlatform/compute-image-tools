#!/usr/bin/env python3
# Copyright 2018 Google Inc. All Rights Reserved.
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

"""Disk utility functions for all VM scripts."""

import logging

import utils

from .common import AptGetInstall
try:
  import guestfs
except ImportError:
  AptGetInstall(['python3-guestfs'])
  import guestfs

_STATUS_PREFIX = 'TranslateStatus: '


def log_key_value(key, value):
  """
  Prints the key and value using the format defined by
  Daisy's serial output inspector.

  The format is defined in `daisy/step_wait_for_instances_signal.go`
  """
  print(_STATUS_PREFIX + "<serial-output key:'%s' value:'%s'>" % (key, value))


def get_physical_drives():
  rc, output = utils.Execute(['lsblk', '--noheadings', '--output=NAME',
                  '--paths', '--list', '--nodeps', '-e7'], capture_output=True)
  disks = []
  if rc == 0:
    disks = output.split('\n')
    disks.remove('')
  else:
    logging.info('Warning: Failed to excute \'lsblk\' cmd, '
                 'Continuing anyway assuming that there are only two disks.')
    disks = ['/dev/sda', '/dev/sdb']

  return disks


def MountDisks(disks) -> guestfs.GuestFS:
  # All new Python code should pass python_return_dict=True
  # to the constructor.  It indicates that your program wants
  # to receive Python dicts for methods in the API that return
  # hashtables.
  g = guestfs.GuestFS(python_return_dict=True)
  # Set the product name as cloud-init checks it to confirm this is a VM in GCE
  g.config('-smbios',
           'type=1,product=Google Compute Engine,manufacturer=Google')

  g.set_memsize(4096)

  # Enable network
  g.set_network(True)

  # Attach the disk image to libguestfs.
  for disk in disks:
    g.add_drive_opts(disk)

  # Run the libguestfs back-end.
  g.launch()

  # Ask libguestfs to inspect for operating systems.
  roots = g.inspect_os()
  if len(roots) == 0:
    raise Exception('inspect_vm: no operating systems found')

  # Sort keys by length, shortest first, so that we end up
  # mounting the filesystems in the correct order.
  mps = g.inspect_get_mountpoints(roots[0])

  g.gcp_image_distro = g.inspect_get_distro(roots[0])
  g.gcp_image_major = str(g.inspect_get_major_version(roots[0]))
  g.gcp_image_minor = str(g.inspect_get_minor_version(roots[0]))

  for device in sorted(list(mps.keys()), key=len):
    try:
      g.mount(mps[device], device)
    except RuntimeError as msg:
      logging.warn('%s (ignored)' % msg)

  return g


def UnmountDisk(g):
  try:
    g.umount_all()
  except Exception as e:
    logging.debug(str(e))
    logging.warn('Unmount failed. Continuing anyway.')
