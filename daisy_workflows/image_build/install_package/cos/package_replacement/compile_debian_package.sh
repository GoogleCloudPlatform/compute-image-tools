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

# This script is responsible for generating a list of binaries and their
# installation paths for the guest agent package. This will be consumed by the
# replacement algorithm (dev_cloudbuild.yaml) that will be responsible for
# performing the replacement process for the guest agent package in COS images.
# The script does the following:
#    1) Downloads the COS specific patches for guest agent (based on COS
#    milestone).
#    2) Downloads the guest agent repository and compiles the debian packaging.
#    3) Parses through the .deb file to determine which files should be replaced
#    (skip licenses).
#
# Most of the code here is sourced from: https://github.com/GoogleCloudPlatform/guest-test-infra/blob/master/packagebuild/daisy_startupscript_deb.sh
#
# Args: ./compile_debian_package [guest_agent_version]
# Example: ./compile_debian_package 094ef227ddf92165abcb7b1241ca44728c3086d1
#     $1 [commit_sha]: the guest agent commit sha (to upgrade to).
#     $2 [arch]: the architecture (amd or arm).

set -o errexit
set -o pipefail
set -o nounset

apply_patches() {
  # Download the repositories and apply COS specific patches. NOTE: The patches will be
  # pulled from the master branch in COS. This is based on the assumption that master
  # will have the most recent guest agent version, and therefore the most recent patches.
  logger -p daemon.info "\nATTENTION: Downloading the board-overlays and guest-agent repos...\n"
  git clone https://cos.googlesource.com/cos/overlays/board-overlays --branch master
  git clone https://github.com/GoogleCloudPlatform/guest-agent.git
  cd guest-agent
  git checkout ${commit_sha}
  cd ..
  mv ./board-overlays/project-lakitu/app-admin/google-guest-agent/files ./guest-agent
  cd guest-agent

  logger -p daemon.info "\nATTENTION: Applying patches...\n"
  search_dir=./files
  for file in "$search_dir"/*
  do
    if ! [[ "$file" == *"homedir"* ]] && [[ "$file" == *"patch"* ]]; then
      git apply "$file"
    fi
  done
}

install_build_deps() {
  # Install the build dependencies.
  logger -p daemon.info "\nATTENTION: Installing build deps...\n"
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
  logger -p daemon.info "\nATTENTION: Finding the debian version...\n"
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
  logger -p daemon.info "\nATTENTION: Installing go...\n"
  cd ..
  source ./install_go.sh; install_go /tmp/go
}

upstream_tarball() {
  # Create the build directory and the upstream tarball.
  # $VERSION is an arbitrary value used for tarball naming.
  # It can be formatted as any date (2 digit): [month][day][year] eg. 010124
  cd guest-agent
  mkdir /tmp/debpackage
  export BUILD_DIR="/tmp/debpackage"
  export VERSION="041224"
  export TAR="${PKGNAME}_${VERSION}.orig.tar.gz"
  logger -p daemon.info "\nATTENTION: Creating build dir and upstream tarball...\n"
  tar czvf "${BUILD_DIR}/${TAR}" --exclude .git --exclude packaging \
    --transform "s/^\./${PKGNAME}-${VERSION}/" .
}

build_tarball(){
  # Extract and build the tarball.
  logger -p daemon.info "\nATTENTION: Extracting tarball and building...\n"
  tar -C "$BUILD_DIR" -xzvf "${BUILD_DIR}/${TAR}"
  cp -r packaging/debian "${BUILD_DIR}/${PKGNAME}-${VERSION}/"
  cd "${BUILD_DIR}/${PKGNAME}-${VERSION}"
  [[ -f debian/changelog ]] && rm debian/changelog
  export RELEASE="g1${DEB}"
  dch --create -M -v 1:${VERSION}-${RELEASE} --package $PKGNAME -D stable \
    "Debian packaging for ${PKGNAME}"
  sudo rm -rf /etc/default/instance_configs.cfg
  debuild -us -uc
  cd ..
}

unpack_binaries(){
  # Unpack the binaries into a local directory that contains 1) the .deb files
  # (which provide installation paths) and 2) binaries in a localized dir format.
  # .deb file naming convention follows that in "upstream_tarball" function.
  # g111 is an arbitrary value given to the .deb file.
  logger -p daemon.info "\nATTENTION: Unpacking binaries in a directory...\n"
  dpkg-deb -x google-guest-agent_${VERSION}-g111_${arch}64.deb debian_binaries
  dpkg-deb -c google-guest-agent_${VERSION}-g111_${arch}64.deb >> google-guest-agent-readable-deb.txt
  logger -p daemon.info "\nATTENTION: google-guest-agent.deb file contents below...\n"
  cat google-guest-agent-readable-deb.txt
  mv debian_binaries /files/package_replacement
  mv google-guest-agent-readable-deb.txt /files/package_replacement
}

identify_replacement_files(){
  logger -p daemon.info "\nATTENTION: Creating a list of files to replace...\n"
  file="/files/package_replacement/google-guest-agent-readable-deb.txt"
  repl_file="repl_files.txt"

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
  mv $repl_file /files/package_replacement
}

main() {
  if [ "$#" -ne 2 ]; then
    echo "Argument 'guest_agent_version' must be provided."
    exit 1
  fi

  commit_sha=$1
  arch=$2

  logger -p daemon.info "\nATTENTION: Starting compile_debian_package.sh...\n"
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
