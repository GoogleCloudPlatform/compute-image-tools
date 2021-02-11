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

"""Run a process in a guestfs.GuestFS instance.

This library addresses a shortcoming of GuestFS.command and GuestFS.sh
that cause data loss. When the command is successful, stderr is discarded.
When the command fails, stdout is discarded.

This library is motivated by subprocess.run, which returns an object
containing stdout, stderr, and the return code of the process.
"""

import logging
import os
import shlex
import textwrap
import typing


def run(g: 'GuestFSInterface', command,
        check: bool = True) -> 'CompletedProcess':
  """Runs a process in a mounted GuestFS instance, ensuring that
  standard output and standard error is always retained.

  Args:
    g: Mounted GuestFS instance.
    command (str or List[str]): Script content that will be executed
    by a bash interpeter on the guest.
    check (bool): When true and the process exits with a non-zero exit code,
    a RuntimeError exception will be raised, using standard error as its
    message. The process's stdout and stderr are written to logging.debug.

  Examples:
    >>> run(g, 'date').stdout
    Thu 05 Nov 2020 06:53:55 PM PST

    >>> run(g, 'printf hi; exit 1', check=False))
    {'stdout': 'hi', 'stderr': '', 'code': 1, 'cmd': 'printf hi; exit 1'}
  """
  tmp_dir = g.mkdtemp('/tmp/gprocXXXXXX')
  program_path = os.path.join(tmp_dir, 'program.sh')
  stdout_path = os.path.join(tmp_dir, 'stdout.txt')
  stderr_path = os.path.join(tmp_dir, 'stderr.txt')
  return_code_path = os.path.join(tmp_dir, 'return_code.txt')

  if not isinstance(command, str):
    command = ' '.join(shlex.quote(s) for s in command)

  program = _make_wrapping_program(command, stdout_path, stderr_path,
                                   return_code_path)

  if check:
    logging.debug('Running %s', command)

  g.write(program_path, program)
  g.command(['/bin/bash', program_path])

  p = CompletedProcess(cmd=command,
                       stdout=g.cat(stdout_path),
                       stderr=g.cat(stderr_path),
                       code=int(g.cat(return_code_path)))
  if check and p.code != 0:
    logging.debug(p)
    raise RuntimeError(p.stderr)
  return p


def _make_wrapping_program(
    cmd: str, stdout_path: str, stderr_path: str,
    return_code_path: str) -> str:
  """Creates a shell script that captures the stdout, stderr, and return code
  to files on the filesystem.

  Args:
    cmd: Command to execute.
    stdout_path: Path to file for stdout.
    stderr_path: Path to file for stderr.
    return_code_path: Path to write return code of executing cmd.

  Returns:
    String containing the shell script.
  """
  return textwrap.dedent("""
  touch {stdout_path}
  touch {stderr_path}
  touch {return_code_path}

  bash -c {cmd} 1> {stdout_path} 2> {stderr_path}
  echo $? > {return_code_path}
  exit 0
  """.format(cmd=shlex.quote(cmd),
             stdout_path=shlex.quote(stdout_path),
             stderr_path=shlex.quote(stderr_path),
             return_code_path=shlex.quote(return_code_path)))


class GuestFSInterface:
  # The subset of guestfs.GuestFS that's used by this module.
  def cat(self, path: str) -> str:
    raise NotImplementedError()

  def command(self, arguments: typing.List[str]) -> str:
    raise NotImplementedError()

  def mkdtemp(self, tmpl: str) -> str:
    raise NotImplementedError()

  def write(self, path: str, content: str):
    raise NotImplementedError()


class CompletedProcess:
  def __init__(self, stdout: str, stderr: str, code: int, cmd: str):
    self.stdout = stdout
    self.stderr = stderr
    self.code = code
    self.cmd = cmd

  def __eq__(self, o: object) -> bool:
    return (isinstance(o, CompletedProcess)
            and o.stdout == self.stdout
            and o.stderr == self.stderr
            and o.code == self.code
            and o.cmd == self.cmd)

  def __repr__(self):
    return str(self.__dict__)
