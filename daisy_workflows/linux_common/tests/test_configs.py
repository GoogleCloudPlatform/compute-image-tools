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

from textwrap import dedent

from utils import configs


def test_comment_all_occurrences():
  original = dedent("""
  ARGS=1,2
  ARGS=3,4
  """).strip()

  expected = dedent("""
  # Removed to support booting on Google Compute Engine.
  # ARGS=1,2
  # Removed to support booting on Google Compute Engine.
  # ARGS=3,4
  # Added to support booting on Google Compute Engine.
  ARGS=10
  """).strip()

  actual = configs.update_grub_conf(original, ARGS='10')

  assert actual == expected


def test_quote_values_with_spaces():
  expected = dedent("""
  # Added to support booting on Google Compute Engine.
  ARGS='a=b, c=d'
  """).strip()

  assert configs.update_grub_conf('', ARGS='a=b, c=d') == expected
