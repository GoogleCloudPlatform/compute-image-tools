#!/bin/bash
# Copyright 2018 Google Inc. All Rights Reserved.
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

source packaging/common.sh 

working_dir=${PWD}

sudo apt-get update

# DEB creation tools.
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install debhelper devscripts build-essential curl tar

# Build dependencies.
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install dh-golang dh-systemd golang-go

dpkg-checkbuilddeps packaging/debian/control

[[ -d /tmp/debpackage ]] && rm -rf /tmp/debpackage
mkdir /tmp/debpackage
tar czvf /tmp/debpackage/${NAME}_${VERSION}.orig.tar.gz --exclude .git --exclude packaging --transform "s/^\./${NAME}-${VERSION}/" .

pushd /tmp/debpackage
tar xzvf ${NAME}_${VERSION}.orig.tar.gz

cd ${NAME}-${VERSION}

cp -r ${working_dir}/packaging/debian ./
cp -r ${working_dir}/*.service ./debian/

debuild -e "VERSION=${VERSION}" -us -uc

popd
