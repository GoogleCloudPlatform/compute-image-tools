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
"""Tests functions exposed by the di.model module."""

import unittest

from boot_inspect import model
from compute_image_tools_proto import inspect_pb2


class TestDistro:

  def test_name_lookup_is_case_insensitive(self):
    assert inspect_pb2.Distro.UBUNTU == model.distro_for("ubuntu")
    assert inspect_pb2.Distro.UBUNTU == model.distro_for("Ubuntu")
    assert inspect_pb2.Distro.OPENSUSE == model.distro_for("opensuse")
    assert inspect_pb2.Distro.OPENSUSE == model.distro_for("openSUSE")
    assert inspect_pb2.Distro.RHEL == model.distro_for("rhel")
    assert inspect_pb2.Distro.RHEL == model.distro_for("RHEL")
    assert inspect_pb2.Distro.CENTOS == model.distro_for("CentOS")
    assert inspect_pb2.Distro.CENTOS == model.distro_for("centos")

  def test_name_lookup_returns_None_when_no_match(self):
    assert model.distro_for("not-a-distro-name") is None
    assert model.distro_for("") is None

  def test_name_lookup_supports_hyphens(self):
    assert inspect_pb2.Distro.SLES_SAP == model.distro_for("sles-sap")


class TestVersion(unittest.TestCase):

  def test_split_happy_case(self):
    assert model.Version("14", "04") == model.Version.split("14.04")
    assert model.Version("2008", "3") == model.Version.split("2008.3")
    assert model.Version("15", "") == model.Version.split("15")

  def test_split_empty_input(self):
    assert model.Version("", "") == model.Version.split("")

  def test_split_allows_non_period(self):
    assert model.Version("fuzzy/fossa", "") == \
           model.Version.split("fuzzy/fossa")
    assert "fuzzy/fossa" == str(model.Version.split("fuzzy/fossa"))

  def test_to_string__happy_case(self):
    assert "15.04", str(model.Version("15", "04"))
    assert "2008.3", str(model.Version("2008", "3"))

  def test_to_string__omits_period_when_only_major(self):
    assert "15" == str(model.Version("15"))

  def test_to_string__empty_when_version_empty(self):
    assert "" == str(model.Version(""))
