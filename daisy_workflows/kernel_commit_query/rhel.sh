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

yum -y update
yum -y install git kernel-devel yum-utils rpm-build \
    platform-python-devel bison flex make gcc python3

kernel_pkg=$(rpm -qf /boot/vmlinuz-`uname -r` | sed 's/:.*$//g')
yumdownloader --source $kernel_pkg
srpm=$(find ./ -name '*.src.rpm' -type f)
rpm -ivh $srpm
spec=$(find $HOME/rpmbuild/SPECS/ -name 'kernel.spec')
rpmbuild --nodeps -bp $spec

source_dir=$(find $HOME/rpmbuild/BUILD/kernel-*/ -name 'linux-*' -type d)
rm -rf /files/distro_kernel && mkdir -p /files/distro_kernel
cp -R $source_dir/* /files/distro_kernel

yum -y install yum-plugin-changelog
yum changelog $kernel_pkg > /files/kernel.changelog

python3 kcq.py
