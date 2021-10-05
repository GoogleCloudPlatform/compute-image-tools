#!/usr/bin/env python3
# Copyright 2021 Google Inc. All Rights Reserved.
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

import logging
import typing


class Apt:
  """Facade for operating with apt to faciliate testing."""

  def __init__(self, run):
    """
    Args:
      run: a function that matches the signature of guestfsprocess.run
    """
    self._run = run

  @staticmethod
  def determine_version_to_install(current_version: str,
                                   available_versions: typing.Set[str],
                                   blocked_versions: typing.Set[str]) -> str:
    non_blocked_versions = set()
    for v in available_versions:
      blocked = False
      for b in blocked_versions:
        if v.startswith(b):
          blocked = True
      if not blocked:
        non_blocked_versions.add(v)
    candidate = ''
    for v in non_blocked_versions:
      if v > current_version and v > candidate:
        candidate = v
    logging.debug({
      'available_versions': available_versions,
      'non_blocked_versions': non_blocked_versions,
      'blocked_versions': blocked_versions,
      'candidate': candidate,
    })
    return candidate

  def get_package_version(self, g, package_name: str) -> str:
    """Returns the package version if it is installed, or an empty
    string if not.

    Args:
      g: A mounted GuestFS instance
      package_name: The package to check
    """
    p = self._run(g, ['dpkg', '-s', package_name],
                  raiseOnError=False)
    if p.code != 0:
      return ''
    for line in p.stdout.splitlines():
      if line.startswith('Version: '):
        parts = line.split(':')
        if len(parts) == 2:
          return parts[1].strip()
    logging.debug('could not find version. dpkg output={}'.format(p))
    return ''

  def list_available_versions(self, g, package_name) -> typing.Set[str]:
    """Returns the versions of package_name that are available to install.

    Args:
      g: A mounted GuestFS instance
      package_name: The package to check
    """
    p = self._run(g, ['apt-cache', 'madison', package_name],
                  raiseOnError=False)
    if p.code != 0:
      return set()
    logging.debug(p)
    versions = set()
    for line in p.stdout.splitlines():
      parts = [s.strip() for s in line.split('|')]
      if len(parts) == 3:
        versions.add(parts[1])
      else:
        logging.debug('skipping line; expected format '
                      '"pkg | version | repo" not found. Line={}'.format(line))
    return versions
