# Copyright 2021 Google Inc. All Rights Reserved.
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

from utils.apt import Apt
from utils.guestfsprocess import CompletedProcess


class TestVersionToInstall:

  def test_return_greatest_match(self):
    assert Apt.determine_version_to_install(
      '1', {'3', '2', '1'}, {}) == '3'

  def test_allow_current_version_empty(self):
    assert Apt.determine_version_to_install(
      '', {'3', '2', '1'}, {}) == '3'

  def test_dont_return_blocked(self):
    assert Apt.determine_version_to_install(
      '1', {'3', '2', '1'}, {'3'}) == '2'

  def test_blocklist_uses_prefix_matching(self):
    assert Apt.determine_version_to_install(
      '1', {'2.1', '2.2'}, {'2'}) == ''

  def test_return_empty_when_nothing_newer(self):
    assert Apt.determine_version_to_install(
      '2', {'1.1'}, set()) == ''


class TestGetPackageVersion:

  def test_return_true_when_package_installed(self):
    stdout = dedent("""
    Package: python3
    Status: install ok installed
    Priority: optional
    Section: python
    Installed-Size: 89
    Maintainer: Matthias Klose <doko@debian.org>
    Architecture: amd64
    Multi-Arch: allowed
    Source: python3-defaults
    Version: 3.9.2-3
    Replaces: python3-minimal (<< 3.1.2-2)
    Provides: python3-profiler
    Depends: python3.9 (>= 3.9.2-0~), libpython3-stdlib (= 3.9.2-3)
    Pre-Depends: python3-minimal (= 3.9.2-3)
    Suggests: python3-doc (>= 3.9.2-3), python3-tk (>= 3.9.2-0~),
    Description: interactive high-level object-oriented languag
     Python, the high-level, interactive object oriented language,
     includes an extensive class library with lots of goodies for
     network programming, system administration, sounds and graphics.
     .
     This package is a dependency package, which depends on Debian's default
     Python 3 version (currently v3.9).
    Homepage: https://www.python.org/
    Cnf-Extra-Commands: python
    Cnf-Priority-Bonus: 5""").strip()
    g = None
    a = Apt(_mock_run(['dpkg', '-s', 'python3'],
                      CompletedProcess(stdout, 'stderr', 0, 'cmd')))
    assert a.get_package_version(g, 'python3') == '3.9.2-3'

  def test_return_empty_string_when_version_not_detected(self):
    stdout = dedent("""
    Package: python3
    Cnf-Priority-Bonus: 5""").strip()
    g = None
    a = Apt(_mock_run(['dpkg', '-s', 'python3'],
                      CompletedProcess(stdout, 'stderr', 0, 'cmd')))
    assert a.get_package_version(g, 'python3') == ''

  def test_return_empty_string_when_package_not_installed(self):
    g = None
    a = Apt(_mock_run(['dpkg', '-s', 'cloud-init'],
                      CompletedProcess('stdout', 'stderr', 1, 'cmd')))
    assert a.get_package_version(g, 'cloud-init') == ''


class TestListAvailableVersions:
  g = None

  def test_return_versions_if_available(self):
    g = None
    stdout = '\n'.join([
      'cloud-init | 21.3-1-g6803368d-0ubuntu1~20.04.3 | repo',
      'cloud-init | 20.1-10-g71af48df-0ubuntu5 | repo',
      'cloud-init | 0.7.5-0ubuntu1.23 | repo',
    ])

    a = Apt(_mock_run(['apt-cache', 'madison', 'cloud-init'],
                      CompletedProcess(stdout, '', 0, 'cmd')))
    assert a.list_available_versions(g, 'cloud-init') == {
      '21.3-1-g6803368d-0ubuntu1~20.04.3',
      '20.1-10-g71af48df-0ubuntu5',
      '0.7.5-0ubuntu1.23'}

  def test_return_empty_list_if_package_not_available(self):
    g = None
    a = Apt(_mock_run(
      ['apt-cache', 'madison', 'cloud-init'],
      CompletedProcess(
        'N: Unable to locate package cloud-init', '', 1, 'cmd')))
    assert a.list_available_versions(g, 'cloud-init') == set()

  def test_skip_lines_from_madison_with_unexpected_syntax(self):
    g = None
    stdout = '\n'.join([
      'cloud-init | 21.3-1-g6803368d-0ubuntu1~20.04.3 | repo',
      'cloud-init | extra bar | repo | not-recognized',
      'cloud-init | missing bar',
      'no bar',
    ])

    a = Apt(_mock_run(['apt-cache', 'madison', 'cloud-init'],
                      CompletedProcess(stdout, '', 0, 'cmd')))
    assert a.list_available_versions(g, 'cloud-init') == {
      '21.3-1-g6803368d-0ubuntu1~20.04.3'}


def _mock_run(expected_args, return_value):
  def run(g, args, raiseOnError=False):
    assert args == expected_args
    return return_value

  return run
