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
"""Expose and consume data using file-like operations."""

import abc
import typing

import guestfs


class Filesystem(object, metaclass=abc.ABCMeta):

  @abc.abstractmethod
  def read_utf8(self, path: str) -> str:
    """Reads the contents of path as a utf8-encoded string.

    Raises:
      FileNotFoundError if path doesn't exist, or it isn't a file.
    """
    pass

  @abc.abstractmethod
  def is_file(self, path: str) -> bool:
    """Returns true if path exists and points to a file."""
    pass

  @abc.abstractmethod
  def is_directory(self, path: str) -> bool:
    """Returns true if path exists and points to a directory."""
    pass


class GuestFSFilesystem(Filesystem):
  """A Filesystem that delegates to an offline VM."""

  def __init__(self, g: guestfs.GuestFS):
    """Args:

      g: A guestfs instance that has been mounted.
    """
    self._g = g

  def read_utf8(self, path: str) -> str:
    read = self._g.read_file(path)
    if read:
      return read.decode('utf-8')

  def is_file(self, path: str) -> bool:
    return self._g.is_file(path, followsymlinks=True)

  def is_directory(self, path: str) -> bool:
    return self._g.is_dir(path)


class DictBackedFilesystem(Filesystem):
  """A Filesystem that delegates to a dict.

  The dict is a string-to-string mapping where
  keys are filenames, and values are utf8-encoded
  strings of their content.
  """

  def __init__(self, fs: typing.Mapping[str, str]):
    """Args:

      fs: A dictionary of file names to file content.
    """
    self.fs = fs

  def read_utf8(self, path: str) -> str:
    if path in self.fs:
      return self.fs[path]
    raise FileNotFoundError(path)

  def is_file(self, path: str) -> bool:
    return path in self.fs

  def is_directory(self, path: str) -> bool:
    if not path.endswith('/'):
      path += '/'
    for fs_path in self.fs.keys():
      if fs_path.startswith(path):
        return True
    return False
