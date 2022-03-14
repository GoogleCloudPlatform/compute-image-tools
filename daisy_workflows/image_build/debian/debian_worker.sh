#!/bin/bash
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

# Builds a Debian based image for import, export, and build tasks. Preloads
# dependencies and binaries for these workflows.

echo "BuildStatus: Updating package cache."
apt -y update
if [[ $? -ne 0 ]]; then
  echo "Trying cache update again."
  apt -y update
  if [[ $? -ne 0 ]]; then
    echo "BuildFailed: Apt cache is failing to update."
    exit 1
  fi
fi

APT_PACKAGES="
debootstrap
dosfstools
kpartx
libguestfs-tools
parted
python3-guestfs
python3-netaddr
python3-pip
rsync
tinyproxy
qemu-utils
"

PIP3_PACKAGES="google-api-python-client google-cloud-storage"

echo "BuildStatus: Installing packages."
export DEBIAN_FRONTEND="noninteractive"
apt-get -y install ${APT_PACKAGES}
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Package install failed."
  exit 1
fi

echo "BuildStatus: Installing python3 libraries from pip."
pip3 install -U ${PIP3_PACKAGES}
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: python3 pip library install failed."
  exit 1
fi

echo "BuildStatus: Downloading gce_export."
curl --output /usr/bin/gce_export https://storage.googleapis.com/compute-image-tools/release/linux/gce_export
if [[ $? -ne 0 ]]; then
  echo "BuildFailed: Could not download gce_export."
  exit 1
fi
chmod +x /usr/bin/gce_export

echo "BuildSuccess: Build succeeded."
