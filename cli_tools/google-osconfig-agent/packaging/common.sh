#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
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

NAME="google-osconfig-agent"
VERSION="0.4.2"
GOLANG="go1.11.4.linux-amd64.tar.gz"
GO=/tmp/go/bin/go
GOPATH=/usr/share/gocode

working_dir=${PWD}
if [[ $(basename "$working_dir") != $NAME ]]; then
  echo "Packaging scripts must be run from top of package dir."
  exit 1
fi

# Golang setup
[[ -d /tmp/go ]] && rm -rf /tmp/go
mkdir -p /tmp/go/
echo "Downloading Go"
curl -s "https://dl.google.com/go/${GOLANG}" -o /tmp/go/go.tar.gz
echo "Extracting Go"
tar -C /tmp/go/ --strip-components=1 -xf /tmp/go/go.tar.gz

echo "Pulling dependencies"
sudo su -c "GOPATH=${GOPATH} ${GO} get -d ./..."
sudo su -c "GOOS=windows GOPATH=${GOPATH} ${GO} get -d ./..."