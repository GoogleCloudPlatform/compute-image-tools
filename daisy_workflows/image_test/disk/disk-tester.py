#!/usr/bin/python
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


import utils

utils.AptGetInstall(['python-pip'])
utils.Execute(['pip', 'install', '--upgrade', 'google-api-python-client'])

from googleapiclient import discovery
import oauth2client.client

MM = utils.MetadataManager
MD = None


def main():
  global MD
  global testee

  compute = utils.GetCompute(discovery, oauth2client.client.GoogleCredentials)
  testee = MM.FetchMetadataDefault('testee')
  testee_disk = MM.FetchMetadataDefault('testee_disk')
  MD = MM(compute, testee)

  # Instance can't be running. Wait for it if not already terminated
  while MD.GetInstanceState(testee) != u'TERMINATED':
    time.sleep(5)

  MD.Wait(MD.ResizeDiskGb(testee_disk, 2049))
  print("DiskResized")

  MD.StartInstance(testee)


if __name__ == '__main__':
  utils.RunTest(main)
