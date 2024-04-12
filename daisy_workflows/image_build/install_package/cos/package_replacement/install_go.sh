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
#
# This script installs go, it's a helper script for compiling the debian pkg.

# Installs a specific version of go for compilation, since availability varies
# across linux distributions. Needs curl and tar to be installed.

# Takes an argument for go installation path. Recommended: "/tmp/go".

install_go() {
  local arch="amd64"
  if [[ `uname -m` == "aarch64" ]]; then
    arch="arm64"
  fi

  # See header file for version specifics.
  local GOLANG="go1.21.4.linux-${arch}.tar.gz"
  export GOPATH=/usr/share/gocode
  export GOCACHE=/tmp/.cache

  # Golang setup
  install_path=$1
  [[ -d ${install_path} ]] && rm -rf ${install_path}
  mkdir -p "${install_path}/"
  curl -s "https://dl.google.com/go/${GOLANG}" -o "${install_path}/go.tar.gz"
  tar -C "${install_path}/" --strip-components=1 -xf "${install_path}/go.tar.gz"

  export PATH="${install_path}/bin:${GOPATH}/bin:${PATH}"  # set path for whoever invokes this function.
  export GO="${install_path}/bin/go"  # reference this go explicitly.
}
