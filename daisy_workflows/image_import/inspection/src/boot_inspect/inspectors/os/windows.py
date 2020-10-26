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
import re

from compute_image_tools_proto import inspect_pb2

# Matched against the output of guestfs.inspect_get_product_variant.
# This is required since desktop and server use the same NT versions.
# For example, NT 6.3 is either Windows 2012r2 or Windows 8.1.
_server_pattern = re.compile('server', re.IGNORECASE)
_client_pattern = re.compile('client', re.IGNORECASE)

# Mappings of NT version to marketing versions.
# Source: https://wikipedia.org/wiki/List_of_Microsoft_Windows_versions
_server_versions = {
    (6, 0): ('2008', ''),
    (6, 1): ('2008', 'r2'),
    (6, 2): ('2012', ''),
    (6, 3): ('2012', 'r2'),
    # (10,0) is resolved in code since, since it's used for
    # both Windows 2016 and Windows 2019.
}
_client_versions = {
    (6, 0): ('Vista', ''),
    (6, 1): ('7', ''),
    (6, 2): ('8', ''),
    (6, 3): ('8', '1'),
    (10, 0): ('10', ''),
}


class Inspector:

  def __init__(self, g, root: str):
    """Supports inspecting offline Windows VMs.

    Args:
      g (guestfs.GuestFS): A guestfs instance that has been mounted.
      root: The root used for mounting.
    """
    self._g = g
    self._root = root

  def inspect(self) -> inspect_pb2.OsRelease:
    distro = self._g.inspect_get_distro(self._root)
    if isinstance(distro, str) and 'windows' in distro.lower():
      return _from_nt_version(
          major_nt=self._g.inspect_get_major_version(self._root),
          minor_nt=self._g.inspect_get_minor_version(self._root),
          variant=self._g.inspect_get_product_variant(self._root),
          product_name=self._g.inspect_get_product_name(self._root)
      )


def _from_nt_version(
    variant: str,
    major_nt: int,
    minor_nt: int,
    product_name: str) -> inspect_pb2.OsRelease:
  major, minor = None, None
  nt_version = major_nt, minor_nt
  if _client_pattern.search(variant):
    major, minor = _client_versions.get(nt_version, (None, None))
  elif _server_pattern.search(variant):
    if nt_version in _server_versions:
      major, minor = _server_versions.get(nt_version, (None, None))
    elif nt_version == (10, 0):
      if '2016' in product_name:
        major, minor = '2016', ''
      elif '2019' in product_name:
        major, minor = '2019', ''

  if major is not None and minor is not None:
    return inspect_pb2.OsRelease(
        major_version=major,
        minor_version=minor,
        distro_id=inspect_pb2.Distro.WINDOWS,
    )
