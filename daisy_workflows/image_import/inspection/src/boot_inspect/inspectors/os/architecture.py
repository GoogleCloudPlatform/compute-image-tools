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

from compute_image_tools_proto import inspect_pb2


class Inspector:

  def __init__(self, g, root: str):
    """Identifies the CPU architecture of a mounted GuestFS instance.

    Args:
      g (guestfs.GuestFS): A guestfs instance that has been mounted.
      root: The root used for mounting.
    """
    self._g = g
    self._root = root

  def inspect(self) -> inspect_pb2.Architecture:
    inspected = self._g.inspect_get_arch(self._root)
    if inspected == 'i386':
      return inspect_pb2.Architecture.X86
    elif inspected == 'x86_64':
      return inspect_pb2.Architecture.X64
    return inspect_pb2.Architecture.UNKNOWN
