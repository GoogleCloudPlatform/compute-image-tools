#!/usr/bin/env bash
# Copyright 2022 Google Inc. All Rights Reserved.
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

set -xeu

apt-get update -y && apt-get install -y git linux-source

rm -rf /files/distro_kernel && mkdir -p /files/distro_kernel
tar --strip-components=1 -xvf $(ls /usr/src/linux-source*.tar.xz) \
    -C /files/distro_kernel

pushd /files/distro_kernel
rm -rf {.git,.gitattributes,.gitignore}
popd

kernel_pkg=$(dpkg -S /boot/vmlinuz-`uname -r` | sed 's/:.*$//g')
apt-get changelog $kernel_pkg > /files/kernel.changelog

python3 /files/kcq.py
