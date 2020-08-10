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
    """Identifies the CPU architecture of a mounted GuestFS instance.

    Args:
      g: A guestfs instance that has been mounted.
      root: The root used for mounting.
    """
    self._g = g
    self._root = root

  def inspect(self) -> model.Architecture:
    inspected = self._g.inspect_get_arch(self._root)
    if inspected == 'i386':
      return model.Architecture.x86
    elif inspected == 'x86_64':
      return model.Architecture.x64
    return model.Architecture.unknown
