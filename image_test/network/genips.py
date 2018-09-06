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

import unittest


def GenIPs(base_ip_str, ip_mask):
  # least significant bits affected
  total_ips = 2 ** (32 - int(ip_mask))
  # Use all ip possibilities to verify that the aliasing is working for all
  base_ip = 0
  for n, i in enumerate(base_ip_str.split('.')):
    base_ip += int(i) * 256 ** (3 - n)
  # make sure it starts with the unmasked bits zeroed
  mask = (2 ** (int(ip_mask)) - 1) << (32 - int(ip_mask))
  base_ip = base_ip & mask

  for n in range(total_ips):
    ip = base_ip + n
    this_octets = [(ip / (256 ** (3 - i))) % 256 for i in range(4)]
    ip_str = '.'.join(map(str, this_octets))
    yield ip_str


class GenIPsTest(unittest.TestCase):
  def test_norange(self):
    base_ip = '127.0.0.1'
    ip_mask = '32'
    self.assertEqual(tuple(GenIPs(base_ip, ip_mask)), ('127.0.0.1',))

  def test_range(self):
    base_ip = '127.0.0.2'
    ip_mask = '30'
    self.assertEqual(tuple(GenIPs(base_ip, ip_mask)),
        ('127.0.0.0', '127.0.0.1', '127.0.0.2', '127.0.0.3',))

  def test_other_octets(self):
    base_ip = '1.1.1.1'
    ip_mask = '23'
    expected = ['1.1.0.' + str(i) for i in range(256)] + \
        ['1.1.1.' + str(i) for i in range(256)]
    self.assertEqual(tuple(GenIPs(base_ip, ip_mask)), tuple(expected))


if __name__ == '__main__':
  unittest.main()
