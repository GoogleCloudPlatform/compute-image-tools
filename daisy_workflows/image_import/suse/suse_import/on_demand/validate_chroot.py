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
"""Ensure directories and files are staged in the chrooted guest to match
what's expected by run_in_chroot.sh

Benefits:
1. Avoid symlinks in the chroot that point outside of it.
2. Provide useful error messages prior to running chroot script.
"""

import logging
from pathlib import Path
import typing


def is_file(expected_file: Path, substring='') -> typing.List[str]:
  """Assert that expected_file exists. If substring is provided,
  assert that the file contains it.

  Returns:
    A list of errors found, or empty if successful.
  """
  logger = logging.getLogger('_is_file {}'.format(expected_file))
  errs = []
  try:
    actual_content = expected_file.read_text()
    logger.debug('content: {}'.format(substring))
    if substring and substring not in actual_content:
      errs.append('{content} not found in {fname}'.format(
          content=substring, fname=expected_file))
  except BaseException as e:
    logger.debug(e)
    errs.append('File not found: {}'.format(expected_file))

  return errs


def is_non_empty_dir(expected_dir: Path) -> typing.List[str]:
  """Assert that directory exists, and that it's not empty.

  Returns:
    A list of errors found, or empty if successful.
  """
  errs = []
  if expected_dir.is_dir():
    for child in expected_dir.iterdir():
      if child.is_file() or (
          child.is_dir() and len(is_non_empty_dir(child)) == 0):
        return errs
    errs.append('Directory is empty: {}'.format(expected_dir))
  else:
    errs.append('Directory not found: {}'.format(expected_dir))
  return errs


def check_root(fs_root: Path, runtime_dir: Path,
               check_os_mounts=True) -> typing.List[str]:
  """Assert that the filesystem rooted at fs_root follows
  the layout expected by run_in_chroot.sh.

  Args:
    fs_root: Directory to consider root of filesystem.
    runtime_dir: Location where run_in_chroot.sh expects
                 its runtime dependencies.
    check_os_mounts: Whether to check mounts such as dev, proc, sys.

  Returns:
    A list of errors found, or empty if successful.

  """
  checks = [
      lambda: is_file(
          runtime_dir / 'cloud_product.txt',
          substring='sle-module-public-cloud'),
      lambda: is_file(
          runtime_dir / 'post_convert_packages.txt'),
      lambda: is_file(
          runtime_dir / 'run_in_chroot.sh',
          substring='#!/usr/bin/env bash'),
      lambda: is_non_empty_dir(
          runtime_dir / 'pre_convert_py'
      ),
      lambda: is_non_empty_dir(
          runtime_dir / 'pre_convert_rpm'
      ),
      lambda: is_file(
          fs_root / 'etc/hosts',
          substring='metadata.google.internal'),
      lambda: is_file(
          fs_root / 'etc/resolv.conf',
          substring='google.internal'),
  ]

  if check_os_mounts:
    checks += [
        lambda: is_non_empty_dir(fs_root / 'dev'),
        lambda: is_non_empty_dir(fs_root / 'proc'),
        lambda: is_non_empty_dir(fs_root / 'sys'),
    ]
  errs = []
  for c in checks:
    errs += c()
  return errs
