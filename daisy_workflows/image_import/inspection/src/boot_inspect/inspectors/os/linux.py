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
"""Supports inspecting offline Linux systems."""

import re
import typing

from boot_inspect import model
import boot_inspect.system.filesystems
from compute_image_tools_proto import inspect_pb2


class FileExistenceMatcher:
  """Supports matching based on whether files exist on the filesystem.

  To illustrate this, these are the metadata files included by RHEL and its
  derivatives:
    RHEL:
      /etc/redhat-release
    CentOS:
      /etc/redhat-release
      /etc/centos-release
    Fedora:
      /etc/redhat-release
      /etc/fedora-release

  In this example, for RHEL, the encoding would be:
    require = {/etc/redhat-release}
    disallow = {/etc/centos-release, /etc/fedora-release}
  """

  def __init__(self, require: typing.Iterable[str] = None,
               disallow: typing.Iterable[str] = None):
    """
    Args:
      require: Iterable of absolute paths. All must be present to be
      eligible for matching.
      disallow: Iterable of absolute paths. None may be present to be
      eligible for matching.
    """
    self._require = require if require else []
    self._disallow = disallow if disallow else []

  def matches(self, fs: boot_inspect.system.filesystems.Filesystem) -> bool:
    positive_indicators = (fs.is_file(f) for f in self._require)
    negative_indicators = (fs.is_file(f) for f in self._disallow)
    return all(positive_indicators) and not any(negative_indicators)


class VersionReader:
  """Identifies versions using pre-systemd metadata files. """

  def __init__(self,
               metadata_file: str,
               version_pattern: typing.Pattern):
    """
    Args:
      metadata_file: The file that *must* be present to allow a match.
      version_pattern: A regex pattern that matches the version string within
      the content of `metadata_file`.
    """
    self._metadata_file = metadata_file
    self._version_pattern = version_pattern

  def get_version(self, fs: boot_inspect.system.filesystems.Filesystem) -> str:
    if fs.is_file(self._metadata_file):
      m = self._version_pattern.search(fs.read_utf8(self._metadata_file))
      if m:
        return m.group(0)


class Fingerprint:
  """Identifies a Linux system.

  Matches are performed against the /etc/os-release file,
  with fallback to legacy metadata files.
  """

  def __init__(self,
               distro: inspect_pb2.Distro,
               aliases: typing.Iterable[str] = (),
               fs_predicate: FileExistenceMatcher = None,
               version_reader: VersionReader = None):
    """
    Args:
      distro: The Distro corresponding to this fingerprint. The 'value' is
      used for matching within the /etc/os-release file.
      aliases: Additional names that indicate a match.
      fs_predicate: Additional predicate that looks at which files are
      present on the system.
      version_reader: Allow version extraction for systems that use
      metadata files other than /etc/os-release.
    """
    self.distro = distro
    self._name_matcher = re.compile(
        '|'.join([
            re.escape(w)
            for w in list(aliases) + [inspect_pb2.Distro.Name(distro)]
        ]), re.IGNORECASE)
    self._fs_predicate = fs_predicate
    self._legacy_version_reader = version_reader

  def _get_version(
      self, etc_os_release: typing.Mapping[str, str],
      fs: boot_inspect.system.filesystems.Filesystem) -> model.Version:

    systemd_version, legacy_version = model.Version(''), model.Version('')

    if 'VERSION_ID' in etc_os_release:
      systemd_version = model.Version.split(etc_os_release['VERSION_ID'])

    if self._legacy_version_reader:
      legacy_version = self._legacy_version_reader.get_version(fs)
      if legacy_version:
        legacy_version = model.Version.split(legacy_version)

    # The assumption here is that the longer string is better.
    # For an example, look at test-data/docker-image-debian:8.8.yaml.
    # In /etc/os-release, the version is '8', while in /etc/debian_version
    # the version is 8.8.
    if str(systemd_version) > str(legacy_version):
      return systemd_version
    return legacy_version

  def match(
      self,
      fs: boot_inspect.system.filesystems.Filesystem) -> inspect_pb2.OsRelease:
    """Returns the OperatingSystem that is identified."""
    etc_os_rel = {}
    if fs.is_file('/etc/os-release'):
      etc_os_rel = _parse_config_file(fs.read_utf8('/etc/os-release'))

    if 'ID' in etc_os_rel:
      matches = self._name_matcher.fullmatch(etc_os_rel['ID']) is not None
      if self._fs_predicate:
        matches &= self._fs_predicate.matches(fs)
    elif self._fs_predicate:
      matches = self._fs_predicate.matches(fs)
    else:
      matches = False

    if matches:
      version = self._get_version(etc_os_rel, fs)
      return inspect_pb2.OsRelease(
          major_version=version.major,
          minor_version=version.minor,
          distro_id=self.distro,
      )


class Inspector:
  """Supports offline inspection of Linux systems."""

  def __init__(self, fs: boot_inspect.system.filesystems.Filesystem,
               fingerprints: typing.List[Fingerprint]):
    """
    Args:
      fs: The filesystem to search.
      fingerprints: The fingerprints to check, which are searched in order.
    """
    self._fs = fs
    self._fingerprints = fingerprints

  def inspect(self) -> inspect_pb2.OsRelease:
    """Returns the OperatingSystem that is identified, or None if a
    match isn't found.
    """
    for d in self._fingerprints:
      match = d.match(self._fs)
      if match:
        return match


def _parse_config_file(content: str) -> typing.Mapping[str, str]:
  """Parses an ini-style config file into a map.

  Each line is expected to be a key/value pair, using `=` as a
  delimiter. Lines without `=` are silently dropped.
  """
  kv = {}
  for line in content.splitlines():
    if not line or line.startswith('#') or '=' not in line:
      continue
    k, v = line.split('=', 1)
    kv[k] = v.strip().strip('"\'')
  return kv
