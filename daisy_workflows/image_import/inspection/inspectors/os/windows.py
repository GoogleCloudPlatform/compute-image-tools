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

import guestfs
import model


class Inspector:

  def __init__(self, g: guestfs.GuestFS, root: str):
    """Supports inspecting offline Windows VMs.

    Args:
      g: A guestfs instance that has been mounted.
      root: The root used for mounting.
    """
    self._g = g
    self._root = root

  def inspect(self) -> model.OperatingSystem:
    if self._is_windows():
      return model.OperatingSystem(
        model.Distro.WINDOWS,
        self._get_version(),
      )

  def _get_version(self) -> model.Version:
    major = self._g.inspect_get_major_version(self._root)
    minor = self._g.inspect_get_minor_version(self._root)
    return model.Version(major, minor)

  def _is_windows(self) -> bool:
    inspected = self._g.inspect_get_distro(self._root)
    return model.distro_for(inspected) == model.Distro.WINDOWS
