#!/bin/bash
i# Copyright 2017 Google Inc. All Rights Reserved.
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
set -x

BYTES_1GB=1073741824
METADATA_URL="http://metadata/computeMetadata/v1/instance"
DISK_DATA_NAME=$(curl -f -H Metadata-Flavor:Google ${METADATA_URL}/attributes/disk-data-name)
SIZE=$(curl -f -H Metadata-Flavor:Google ${METADATA_URL}/attributes/disk-size)
IMAGE_DESTINATION=$(curl -f -H Metadata-Flavor:Google ${METADATA_URL}/attributes/image-destination)
ZONE=$(curl -f -H Metadata-Flavor:Google ${METADATA_URL}/zone | cut -d'/' -f4)
RESERVED_SPACE=50 #50MB reserved space so system can still run

echo "GCECreateDisk: Checking whether image has been created."
IMAGE_URI=$(gcloud compute images list --project compute-image-tools-test --uri --filter="name=(${IMAGE_DESTINATION})")
echo "Image uri: ${IMAGE_URI}"
if [[ ${IMAGE_URI} == https* ]]; then
  echo "CreateDiskFailed: Image ${IMAGE_DESTINATION} exists. Skip creating image."
  exit
fi

echo "GCECreateDisk: Fullfilling disk of size ${SIZE}GB."
SIZE_MB=$(awk "BEGIN {print int(${SIZE} * 1024)}")
sudo mkfs.ext4 /dev/sdb
sudo mkdir -p /mnt/sdb
sudo mount /dev/sdb /mnt/sdb
sudo dd if=/dev/zero of=/mnt/sdb/reserved.space count=${RESERVED_SPACE} bs=1M
sudo dd if=/dev/urandom of=/mnt/sdb/random.file count=${SIZE_MB} bs=1M
sudo rm /mnt/sdb/reserved.space

echo "create disk success"
sync
