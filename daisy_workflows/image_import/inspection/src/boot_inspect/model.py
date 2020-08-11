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

import enum
import json
import re
import typing


class Distro(enum.Enum):
  """Distributions that are detectable.

  The value of the enum is conveyed to the user.
  """
  AMAZON = 'amazon'
  CENTOS = 'centos'
  DEBIAN = 'debian'
  FEDORA = 'fedora'
  KALI = 'kali'
  OPENSUSE = 'opensuse'
  ORACLE = 'oracle'
  RHEL = 'rhel'
  SLES = 'sles'
  UBUNTU = 'ubuntu'
  WINDOWS = 'windows'


def distro_for(name: str):
  """Returns the Distro that corresponds to `name`.

  The search is case insensitive, and is performed against the enum's value.
  None is returned when there is no match.
  """
  if not name:
    return None
  pattern = re.compile(name, re.IGNORECASE)
  for d in Distro:
    if pattern.fullmatch(d.value):
      return d


class Architecture(enum.Enum):
  x64 = 'x64'
  x86 = 'x86'
  unknown = 'unknown'


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


class OperatingSystem:
  """Encapsulates a specific operating system release.

  Examples:
    Windows 6.1
    Ubuntu 14.04
    CentOS 6
  """

  def __init__(self, distro: Distro, version: Version):
    self.distro = distro
    self.version = version

  def __repr__(self) -> str:
    return str(self.__dict__)

  def __eq__(self, other):
    if isinstance(other, self.__class__):
      return self.__dict__ == other.__dict__
    else:
      return False


class InspectionResults:
  """Collection of all inspection results."""

  def __init__(self, device: str, os: OperatingSystem,
               architecture: Architecture):
    self.device = device
    self.os = os
    self.architecture = architecture


class ModelJSONEncoder(json.JSONEncoder):
  """Supports JSON encoding of the classes defined in this module."""

  def default(self, o: typing.Any) -> typing.Any:
    if isinstance(o, enum.Enum):
      return o.value
    if isinstance(o, Version):
      return o.__dict__
    if isinstance(o, OperatingSystem):
      return o.__dict__
    if isinstance(o, InspectionResults):
      return o.__dict__
    return super().default(o)
