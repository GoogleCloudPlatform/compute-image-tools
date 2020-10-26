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
"""Domain objects related to disk inspection."""

import re

from compute_image_tools_proto import inspect_pb2


def distro_for(name: str) -> inspect_pb2.Distro:
  """Returns the Distro that corresponds to `name`.

  The search is case insensitive, and is performed against the enum's value.
  None is returned when there is no match.
  """
  if not name:
    return None
  pattern = re.compile(name.replace('-', '_'), re.IGNORECASE)
  for d in inspect_pb2.Distro.DESCRIPTOR.values_by_name:
    if pattern.fullmatch(d):
      return inspect_pb2.Distro.Value(d)


class Version:
  """Representation of a release version.

  Supports versions schemes with a major number, and an optional minor
  number. For example:
   * CentOS 6:   major 6, minor <empty>
   * CentOS 6.0: major 6, minor 0
  """

  def __init__(self, major: str, minor: str = ''):
    self.major = str(major) if major else ''
    self.minor = str(minor) if minor else ''

  @staticmethod
  def split(version: str) -> 'Version':
    """Creates a `Version` instance from a pre-encoded version string.

    Args:
      version: A version encoded as {major}.{minor} or {major}.
    """
    if '.' in version:
      major, minor = version.split('.', 1)
      return Version(major, minor)
    return Version(major=version)

  def __repr__(self) -> str:
    if self.minor:
      return '%s.%s' % (self.major, self.minor)
    return self.major

  def __eq__(self, o: object) -> bool:
    return (isinstance(o, Version)
            and o.major == self.major
            and o.minor == self.minor)
