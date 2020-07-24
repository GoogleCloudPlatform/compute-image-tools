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
"""Perform inspection, and print results to stdout.

Inspection results are emitted using the key-value format
supported by Daisy's serial log collector.

The first argument to the script is the device to inspect.
"""

import sys

import inspection


def kv(key: str, value: str):
  template = "Status: <serial-output key:'{key}' value:'{value}'>"
  return template.format(key=key, value=value)


def main(device: str):
  results = inspection.inspect_device(device)
  if results:
    print(kv('architecture', results.architecture.value))
    print(kv('distro', results.os.distro.value))
    print(kv('major', results.os.version.major))
    print(kv('minor', results.os.version.minor))
  print('Success: Done!')


if __name__ == '__main__':
  if len(sys.argv) != 2:
    print('Usage: ./daisy_shim.py <block device | disk file>')
    sys.exit(1)
  device = sys.argv[1]
  main(device)
