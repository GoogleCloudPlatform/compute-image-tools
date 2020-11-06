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
import pathlib
import subprocess
import tempfile
from unittest.mock import MagicMock

from utils.guestfsprocess import CompletedProcess, GuestFSInterface, run


def test_capture_stdout():
  cmd = 'echo abc123'
  result = run(_make_local_guestfs(), cmd)
  assert result == CompletedProcess('abc123\n', '', 0, cmd)


def test_capture_stderr():
  cmd = 'echo error msg > /dev/stderr'
  result = run(_make_local_guestfs(), cmd)
  assert result == CompletedProcess('', 'error msg\n', 0, cmd)


def test_support_positive_code():
  cmd = 'exit 100'
  result = run(_make_local_guestfs(), cmd)
  assert result == CompletedProcess('', '', 100, cmd)


def test_support_array_args():
  result = run(_make_local_guestfs(), ['echo', 'hi'])
  assert result == CompletedProcess('hi\n', '', 0, 'echo hi')


def test_escape_array_members():
  result = run(_make_local_guestfs(), ['echo', 'hello', '; ls *'])
  assert result == CompletedProcess('hello ; ls *\n', '', 0,
                                    "echo hello '; ls *'")


def test_capture_runtime_errors():
  result = run(_make_local_guestfs(), 'not-a-command')
  assert result.code != 0
  assert 'not-a-command' in result.stderr


def test_capture_output_when_non_zero_return():
  cmd = 'printf content; printf err > /dev/stderr; exit 1'
  result = run(_make_local_guestfs(), cmd)
  assert result == CompletedProcess('content', 'err', 1, cmd)


def _make_local_guestfs():
  tmp_dir = tempfile.mkdtemp()
  g = GuestFSInterface()
  g.mkdtemp = MagicMock(return_value=tmp_dir)
  g.cat = lambda path: pathlib.Path(path).read_text()
  g.command = lambda args: subprocess.run(args, check=True)
  g.write = lambda path, txt: pathlib.Path(path).write_text(txt)
  return g
