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
#
set -e

NAME="google-osconfig-agent"
VERSION="0.4.1"

working_dir=${PWD}
if [[ $(basename "$working_dir") != $NAME ]]; then
  echo "Packaging scripts must be run from top of package dir."
  exit 1
fi

# DEB creation tools.
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install debhelper devscripts build-essential curl tar

# Build dependencies.
DEBIAN_FRONTEND=noninteractive sudo apt-get -y install dh-golang dh-systemd

dpkg-checkbuilddeps packaging/debian/control

# Golang11 setup
[[ -d /tmp/go11 ]] && rm -rf /tmp/go11
mkdir -p /tmp/go11/
echo "Downloading Go 11"
curl -s https://dl.google.com/go/go1.11.4.linux-amd64.tar.gz -o /tmp/go11/go.tar.gz
echo "Extracting Go 11"
tar -C /tmp/go11/ --strip-components=1 -xf /tmp/go11/go.tar.gz

# Go Build dependencies
sudo su -c 'GOPATH=/usr/share/gocode /tmp/go11/bin/go get -d ./...'

[[ -d /tmp/debpackage ]] && rm -rf /tmp/debpackage
mkdir /tmp/debpackage
tar czvf /tmp/debpackage/${NAME}_${VERSION}.orig.tar.gz  --exclude .git --exclude packaging --transform "s/^\./${NAME}-${VERSION}/" .

pushd /tmp/debpackage
tar xzvf ${NAME}_${VERSION}.orig.tar.gz

cd ${NAME}-${VERSION}

cp -r ${working_dir}/packaging/debian ./
cp -r ${working_dir}/*.service ./debian/

debuild -us -uc

popd
