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

from pathlib import Path
import re
import tempfile
import typing

from on_demand import validate_chroot


def test_validate_chroot():
  fs_root = Path('tests/test-data/validate_chroot')
  runtime_path = fs_root / 'runtime'
  errs = validate_chroot.check_root(fs_root, runtime_path,
                                    check_os_mounts=False)
  assert len(errs) == 0


class TestIsFile:

  def test_is_file_validates_existence(self):
    f = Path(tempfile.mkdtemp()) / 'f.txt'
    assert not f.exists()
    errs = validate_chroot.is_file(f)
    assert len(errs) == 1
    assert contains(errs, 'File not found.*f.txt')

    f.touch()
    assert f.exists()
    errs = validate_chroot.is_file(f)
    assert len(errs) == 0

  def test_is_file_validates_substring(self):
    f = Path(tempfile.mkdtemp()) / 'f.txt'
    f.touch()
    errs = validate_chroot.is_file(f, substring='xyz')
    assert len(errs) == 1
    assert contains(errs, 'xyz not found')

    f.write_text('hello xyz')
    errs = validate_chroot.is_file(f, substring='xyz')
    assert len(errs) == 0


class TestIsNonEmptyDir:

  def test_is_non_empty_dir_passing(self):
    d = Path(tempfile.mkdtemp()) / 'logs/2020'
    d.mkdir(parents=True)
    (d / 'file').touch()
    errs = validate_chroot.is_non_empty_dir(d.parent)
    assert len(errs) == 0

  def test_is_non_empty_dir_validates_existence(self):
    d = Path(tempfile.mkdtemp()) / 'logs'
    errs = validate_chroot.is_non_empty_dir(d)
    assert len(errs) == 1
    assert contains(errs, 'Directory not found.*/logs')

  def test_is_non_empty_dir_requires_file(self):
    d = Path(tempfile.mkdtemp()) / 'logs'
    d.mkdir()
    # A child *directory* shouldn't allow the test to pass.
    (d / 'file').mkdir()
    errs = validate_chroot.is_non_empty_dir(d)
    assert len(errs) == 1
    assert contains(errs, 'Directory is empty.*/logs')


def contains(arr: typing.List[str], pattern: str):
  """Passes if at least one element of arr contain the pattern."""
  p = re.compile(pattern)
  for e in arr:
    if p.search(e):
      return True
  return False
