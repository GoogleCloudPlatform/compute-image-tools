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

import model
import system.filesystems


class LegacyFingerprint:
  """Identifies systems using pre-systemd metadata files.

  Prior to systemd's specification of /etc/os-release,
  https://www.freedesktop.org/software/systemd/man/os-release.html,
  there was no universal standard for specifying system metadata. This
  class encodes which files were used during that period.

  A positive match is based on:
    1. The existence of a standard metadata file
    2. The non-existance of derivative metadata files.

  To illustrate this, these are the metadata files included by RHEL and its
  derivatives:
    RHEL:
      /etc/redhat-release
    CentOS:
      /etc/redhat-release
      /etc/centos-release
    CentOS:
      /etc/redhat-release
      /etc/fedora-release

  In this example, for RHEL, the encoding would be:
    metadata_file = /etc/redhat-release
    derivative_metadata_files = /etc/centos-release, /etc/fedora-release
  """

  def __init__(self,
               metadata_file: str,
               version_pattern: typing.Pattern,
               derivative_metadata_files: typing.Iterable[str] = ()):
    """
    Args:
      metadata_file: The file that *must* be present to allow a match.
      version_pattern: A regex pattern that matches the version string within
      the content of `metadata_file`.
      derivative_metadata_files: Files that indicate the presence of a
      derivative distro.
    """
    self._metadata_file = metadata_file
    self._version_pattern = version_pattern
    self._derivatives = derivative_metadata_files

  def get_version(self, fs: system.filesystems.Filesystem) -> str:
    if fs.is_file(self._metadata_file):
      m = self._version_pattern.search(fs.read_utf8(self._metadata_file))
      if m:
        return m.group(0)

  def matches(self, fs: system.filesystems.Filesystem) -> bool:
    for anti in self._derivatives:
      if fs.is_file(anti):
        return False
    return fs.is_file(self._metadata_file)


class Fingerprint:
  """Identifies a Linux system.

  Matches are performed against the /etc/os-release file,
  with fallback to legacy metadata files.
  """

  def __init__(self,
               distro: model.Distro,
               aliases: typing.Iterable[str] = (),
               legacy: LegacyFingerprint = None):
    """
    Args:
      distro: The Distro corresponding to this fingerprint. The 'value' is
      used for matching within the /etc/os-release file.
      aliases: Additional names that indicate a match.
      legacy: Configuration for systems that require inspection beyond
      /etc/os-release.
    """
    self.distro = distro
    self._name_matcher = re.compile(
      '|'.join([re.escape(w) for w in list(aliases) + [distro.value]]),
      re.IGNORECASE)
    self._legacy = legacy

  def _get_version(self, etc_os_release: typing.Mapping[str, str],
                   fs: system.filesystems.Filesystem) -> model.Version:

    systemd_version, legacy_version = model.Version(''), model.Version('')

    if 'VERSION_ID' in etc_os_release:
      systemd_version = model.Version.split(etc_os_release['VERSION_ID'])

    if self._legacy:
      legacy_version = self._legacy.get_version(fs)
      if legacy_version:
        legacy_version = model.Version.split(legacy_version)

    # The assumption here is that the longer string is better.
    # For an example, look at test-data/docker-image-debian:8.8.yaml.
    # In /etc/os-release, the version is '8', while in /etc/debian_version
    # the version is 8.8.
    if str(systemd_version) > str(legacy_version):
      return systemd_version
    return legacy_version

  def match(self, fs: system.filesystems.Filesystem) -> model.OperatingSystem:
    """Returns the OperatingSystem that is identified."""
    etc_os_rel = {}
    if fs.is_file('/etc/os-release'):
      etc_os_rel = _parse_config_file(fs.read_utf8('/etc/os-release'))
    if 'ID' in etc_os_rel:
      matches = self._name_matcher.fullmatch(etc_os_rel['ID']) is not None
    else:
      matches = self._legacy and self._legacy.matches(fs)
    if matches:
      return model.OperatingSystem(
        distro=self.distro,
        version=self._get_version(etc_os_rel, fs),
      )


class Inspector:
  """Supports offline inspection of Linux systems."""

  def __init__(self, fs: system.filesystems.Filesystem,
               fingerprints: typing.List[Fingerprint]):
    """
    Args:
      fs: The filesystem to search.
      fingerprints: The fingerprints to check, which are searched in order.
    """
    self._fs = fs
    self._fingerprints = fingerprints

  def inspect(self) -> model.OperatingSystem:
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
