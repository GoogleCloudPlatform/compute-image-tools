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

from setuptools import setup

setup(
    name="suse_import",
    version="0.1",
    packages=["on_demand"],
    install_requires=[
        "linux_common",
    ],
    py_modules=["translate"],
    package_data={
        "on_demand": ["*.sh"],
    },
    entry_points={
        "console_scripts": [
            "translate-suse = translate:translate",
        ],
    })
