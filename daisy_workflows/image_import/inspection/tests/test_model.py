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

import json
import unittest

from boot_inspect import model


class TestDistro:

  def test_name_lookup_is_case_insensitive(self):
    assert model.Distro.UBUNTU == model.distro_for("ubuntu")
    assert model.Distro.UBUNTU == model.distro_for("Ubuntu")
    assert model.Distro.OPENSUSE == model.distro_for("openSUSE")
    assert model.Distro.RHEL == model.distro_for("RHEL")
    assert model.Distro.CENTOS == model.distro_for("CentOS")
    assert model.Distro.CENTOS == model.distro_for("centos")

  def test_name_lookup_returns_None_when_no_match(self):
    assert model.distro_for("not-a-distro-name") is None
    assert model.distro_for("") is None


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

  def test_repr_happy_case(self):
    assert "15.04", str(model.Version("15", "04"))
    assert "2008.3", str(model.Version("2008", "3"))

  def test_repr_omits_period_when_only_major(self):
    assert "15" == str(model.Version("15"))

  def test_repr_empty_when_version_empty(self):
    assert "" == str(model.Version(""))


class TestJSONEncoder:

  def test_happy_case(self):
    inspection_results = model.InspectionResults(
      device="/dev/sdb",
      os=model.OperatingSystem(
        distro=model.Distro.WINDOWS,
        version=model.Version(major="6", minor="1"),
      ),
      architecture=model.Architecture.x86,
    )

    expected = {
      "device": "/dev/sdb",
      "os": {
        "distro": "windows",
        "version": {
          "major": "6",
          "minor": "1",
        }
      },
      "architecture": "x86",
    }
    actual = json.dumps(inspection_results, cls=model.ModelJSONEncoder)
    assert expected == json.loads(actual)

  def test_allow_empty_minor_version(self):
    inspection_results = model.InspectionResults(
      device="/dev/sdb",
      os=model.OperatingSystem(
        distro=model.Distro.UBUNTU,
        version=model.Version(major="14", ),
      ),
      architecture=model.Architecture.x64,
    )

    expected = {
      "device": "/dev/sdb",
      "os": {
        "distro": "ubuntu",
        "version": {
          "major": "14",
          "minor": "",
        }
      },
      "architecture": "x64",
    }
    actual = json.dumps(inspection_results, cls=model.ModelJSONEncoder)
    assert expected == json.loads(actual)

  def test_keep_leading_zeroes_in_version(self):
    inspection_results = model.InspectionResults(
      device="/dev/sdb",
      os=model.OperatingSystem(
        distro=model.Distro.UBUNTU,
        version=model.Version(major="0x0", minor="04"),
      ),
      architecture=model.Architecture.x64,
    )

    expected = {
      "device": "/dev/sdb",
      "os": {
        "distro": "ubuntu",
        "version": {
          "major": "0x0",
          "minor": "04",
        }
      },
      "architecture": "x64",
    }
    actual = json.dumps(inspection_results, cls=model.ModelJSONEncoder)
    assert expected == json.loads(actual)

  def test_allow_all_fields_empty(self):
    inspection_results = model.InspectionResults(
      os=None, device=None, architecture=None)

    expected = {"architecture": None, "device": None, "os": None}
    actual = json.dumps(inspection_results, cls=model.ModelJSONEncoder)
    assert expected == json.loads(actual)
