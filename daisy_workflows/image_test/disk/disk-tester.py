#!/usr/bin/env python2
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

import time

from google import auth
from googleapiclient import discovery
import utils

MM = utils.MetadataManager
MD = None
key = None


def TestDiskResize(testee, testee_disk):
  # Instance can't be running. Wait for it if not already terminated
  while MD.GetInstanceState(testee) != u'TERMINATED':
    time.sleep(5)

  MD.Wait(MD.ResizeDiskGb(testee_disk, 2049))
  print("DiskResized")


def CheckSdbCmd():
  return ['ls', '/dev/sdb']


def TestDiskAttach(testee, removable_disk):
  MD.StartInstance(testee)

  # test attaching disk while running
  while MD.GetInstanceState(testee) != u'RUNNING':
    time.sleep(5)

  # second disk should not be available
  global key
  key = MD.AddSshKey(MM.SSH_KEYS)
  utils.ExecuteInSsh(key, MD.ssh_user, testee, CheckSdbCmd(), expect_fail=True)

  MD.Wait(MD.AttachDisk(testee, removable_disk))

  # should detect a second disk
  utils.ExecuteInSsh(key, MD.ssh_user, testee, CheckSdbCmd())
  print("DiskAttached")


def TestDiskDetach(testee, removable_disk):
  # test detaching disk
  disk_device_name = MD.GetDiskDeviceNameFromAttached(testee, removable_disk)
  MD.Wait(MD.DetachDisk(testee, disk_device_name))

  # second disk should not be available anymore
  utils.ExecuteInSsh(key, MD.ssh_user, testee, CheckSdbCmd(), expect_fail=True)
  print("DiskDetached")


def main():
  global MD

  credentials, _ = auth.default()
  compute = utils.GetCompute(discovery, credentials)
  testee = MM.FetchMetadataDefault('testee')
  testee_disk = MM.FetchMetadataDefault('testee_disk')
  removable_disk = MM.FetchMetadataDefault('testee_disk_removable')
  MD = MM(compute, testee)

  TestDiskResize(testee, testee_disk)
  TestDiskAttach(testee, removable_disk)
  TestDiskDetach(testee, removable_disk)


if __name__ == '__main__':
  utils.RunTest(main)
