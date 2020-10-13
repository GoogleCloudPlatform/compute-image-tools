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

import os

from boot_inspect import inspection
import pytest
import yaml


@pytest.mark.parametrize("fpath", [
    os.path.join('tests/test-data/gdisk', f) for f in
    os.listdir('tests/test-data/gdisk')
])
def test_yaml_encoded_cases(fpath):
  with open(fpath) as stream:
    loaded_yaml = yaml.safe_load(stream)
    assert 'output' in loaded_yaml
    assert 'expected' in loaded_yaml
    output = loaded_yaml['output']
    expected = loaded_yaml['expected']

  bios_bootable = inspection._inspect_for_hybrid_mbr(output)
  assert expected == bios_bootable
