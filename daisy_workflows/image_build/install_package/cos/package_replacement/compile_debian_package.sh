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

# This script compiles the debian packaging for google guest agent and creates a list of files to replace during COS guest agent package replacement.
# Most of the code here is sourced from: https://github.com/GoogleCloudPlatform/guest-test-infra/blob/master/packagebuild/daisy_startupscript_deb.sh
# Compiling the debian package produces a set of binaries to be installed and their respective installation paths. Then we mark the binaries and
# service files for replacement (skipping licenses for testing purposes).

# Args: ./compile_debian_package [overlays_branch] [guest_agent_version]
# Example: ./compile_debian_package master 20231214.00
#     $1 [overlays_branch]: the COS milestone version (to apply the correct patches).
#     $2 [guest_agent_version]: the guest agent version (to upgrade to).

set -o errexit
set -o pipefail
set -o nounset

apply_patches() {
  # Download the repositories and apply COS specific patches.
  echo -e "\nATTENTION: Downloading the board-overlays and guest-agent repos...\n"
  git clone https://cos.googlesource.com/cos/overlays/board-overlays --branch ${overlays_branch}
  git clone https://github.com/GoogleCloudPlatform/guest-agent.git --branch ${guest_agent_version}
  mv ./board-overlays/project-lakitu/app-admin/google-guest-agent/files ./guest-agent
  cd guest-agent

  echo -e "\nATTENTION: Applying patches...\n"
  search_dir=./files
  for file in "$search_dir"/*
  do
    if [[ "$file" == *"patch"* ]]; then
      git apply "$file"
    fi
  done
}

install_build_deps() {
  # Install the build dependencies.
  echo -e "\nATTENTION: Installing build deps...\n"
  apt-get -y update
  apt-get install -y --no-install-{suggests,recommends} git-core \
    debhelper devscripts build-essential equivs libdistro-info-perl
  export PKGNAME="google-guest-agent"
  mk-build-deps -t "apt-get -o Debug::pkgProblemResolver=yes \
    --no-install-recommends --yes" --install packaging/debian/control
  dpkg-checkbuilddeps packaging/debian/control
}

find_debian_version() {
  # Identify and store the debian version.
  echo -e "\nATTENTION: Finding the debian version...\n"
  cat /etc/debian_version
  DEB_VERSION=$(</etc/debian_version)
  # deb_version=${DEB}
  if [[ "$DEB_VERSION" =~ "bookworm" ]]; then
      export DEB="12"
  else
      export DEB="11"
  fi
}

install_go(){
  # Install the correct version of Go.
  echo -e "\nATTENTION: Installing go...\n"
  cd /workspace
  source ./install_go.sh; install_go /tmp/go
}

upstream_tarball() {
  # Create the build directory and the upstream tarball.
  cd guest-agent
  mkdir /tmp/debpackage
  export BUILD_DIR="/tmp/debpackage"
  export VERSION="112723"
  export TAR="${PKGNAME}_${VERSION}.orig.tar.gz"
  echo -e "\nATTENTION: Creating build dir and upstream tarball...\n"
  tar czvf "${BUILD_DIR}/${TAR}" --exclude .git --exclude packaging \
    --transform "s/^\./${PKGNAME}-${VERSION}/" .
}

build_tarball(){
  # Extract and build the tarball.
  echo -e "\nATTENTION: Extracting tarball and building...\n"
  tar -C "$BUILD_DIR" -xzvf "${BUILD_DIR}/${TAR}"
  cp -r packaging/debian "${BUILD_DIR}/${PKGNAME}-${VERSION}/"
  cd "${BUILD_DIR}/${PKGNAME}-${VERSION}"
  [[ -f debian/changelog ]] && rm debian/changelog
  export RELEASE="g1${DEB}"
  dch --create -M -v 1:${VERSION}-${RELEASE} --package $PKGNAME -D stable \
    "Debian packaging for ${PKGNAME}"
  debuild -us -uc
  cd ..
}

unpack_binaries(){
  # Unpack the binaries into a local directory that contains 1) the .deb files
  # (which provide installation paths) and 2) binaries in a localized dir format.
  echo -e "\nATTENTION: Unpacking binaries in a directory...\n"
  dpkg-deb -x google-guest-agent_112723-g111_amd64.deb debian_binaries
  dpkg-deb -c google-guest-agent_112723-g111_amd64.deb >> google-guest-agent-readable-deb.txt
  echo -e "\nATTENTION: google-guest-agent.deb file contents below...\n"
  cat google-guest-agent-readable-deb.txt
  mv debian_binaries /workspace/upload
  mv google-guest-agent-readable-deb.txt /workspace
}

identify_replacement_files(){
  echo -e "\nATTENTION: Creating a list of files to replace...\n"
  file="/workspace/google-guest-agent-readable-deb.txt"
  repl_file="/workspace/repl_files.txt"

  # Go through every line in the deb file...
  while read -r line; do

    # For every line...
    read -ra arr <<< "$line"

    # Store file permissions and the file path eg "./usr/bin/google_guest_agent"
    permissions="${arr[0]}"
    deb_file_path="${arr[5]}"

    # If the file doesn't have the "drwxr" permissions, is not located under '/usr/share' and the file is not "gce-workload-cert-refresh"...
    if ! [[ "$permissions" == *"drwxr"* ]] && ! [[ "$deb_file_path" == *"/usr/share"* ]] && ! [[ "$deb_file_path" == *"gce-workload-cert-refresh"* ]]; then

      # Write the deb_file_path to a replacement list file.
      echo $deb_file_path >> $repl_file

    fi
  done <$file
  cat $repl_file
}

main() {
  if [ "$#" -ne 2 ]; then
    echo "Arguments 'overlays_branch' and 'guest_agent_version' must be provided."
    exit 1
  fi

  overlays_branch=$1
  guest_agent_version=$2

  echo -e "\nATTENTION: Starting compile_debian_package.sh...\n"
  apply_patches
  install_build_deps
  find_debian_version
  install_go
  upstream_tarball
  build_tarball
  unpack_binaries
  identify_replacement_files
}

main "$@"
