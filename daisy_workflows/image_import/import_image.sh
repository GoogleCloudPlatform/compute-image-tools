#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
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
URL="http://metadata/computeMetadata/v1/instance"
DAISY_SOURCE_URL="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/daisy-sources-path)"
SOURCE_URL="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/source_disk_file)"
DISKNAME="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/disk_name)"
SCRATCH_DISK_NAME="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/scratch_disk_name)"
ME="$(curl -f -H Metadata-Flavor:Google ${URL}/name)"
ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/zone)

SOURCE_FILE_EXT="${SOURCE_URL##*.}"
SOURCE_SIZE_BYTES="$(gsutil du ${SOURCE_URL} | grep -o '^[0-9]\+')"
IMAGE_PATH="/daisy-scratch/image.${SOURCE_FILE_EXT}"

# Print info.
echo "#################" 2> /dev/null
echo "# Configuration #" 2> /dev/null
echo "#################" 2> /dev/null
echo "IMAGE_PATH: ${IMAGE_PATH}" 2> /dev/null
echo "SOURCE_URL: ${SOURCE_URL}" 2> /dev/null
echo "SOURCE_SIZE_BYTES: ${SOURCE_SIZE_BYTES}" 2> /dev/null
echo "SOURCE_FILE_EXT: ${SOURCE_FILE_EXT}" 2> /dev/null
echo "DISKNAME: ${DISKNAME}" 2> /dev/null
echo "ME: ${ME}" 2> /dev/null
echo "ZONE: ${ZONE}" 2> /dev/null

function resizeDisk() {
  local diskId="${1}"
  local newSizeInGb="${2}"
  local zone="${3}"

  echo "Import: Resizing ${diskId} to ${newSizeInGb}GB in ${zone}."
  if ! out=$(gcloud -q compute disks resize ${diskId} --size=${newSizeInGb}GB --zone=${zone} 2>&1); then
    echo "ImportFailed: Failed to resize disk. [Privacy-> resize disk ${diskId} to ${newSizeInGb}GB in ${zone}, error: ${out} <-Privacy]"
    exit
  fi
}

function copyImageToScratchDisk() {
  local scratchDiskSizeGigabytes=$(awk "BEGIN {print int(((${SOURCE_SIZE_BYTES} - 1)/${BYTES_1GB}) + 1)}")
  # We allocate double capacity for OVA, which would
  # require making an additional copy of its enclosed VMDK.
  if [[ "${SOURCE_FILE_EXT}" == "ova" ]]; then
     scratchDiskSizeGigabytes=$((scratchDiskSizeGigabytes * 2))
  fi

  # This disk is initially created with 10GB of space.
  # Enlarge it if that's insufficient to hold the input image.
  if [[ ${scratchDiskSizeGigabytes} -gt 10 ]]; then
    resizeDisk "${SCRATCH_DISK_NAME}" "${scratchDiskSizeGigabytes}" "${ZONE}"
  fi

  mkdir -p /daisy-scratch
  # /dev/sdb is used since the scratch disk is the second
  # disk that's attached in import_disk.wf.json.
  mkfs.ext4 /dev/sdb
  mount /dev/sdb /daisy-scratch
  if [[ $? -ne 0 ]]; then
    echo "ImportFailed: Failed to prepare scratch disk."
  fi
  gsutil cp "${SOURCE_URL}" "${IMAGE_PATH}"
  echo "Import: Copied image from ${SOURCE_URL} to ${IMAGE_PATH}"
}

copyImageToScratchDisk

# If the image is an OVA, then copy out its VMDK.
if [[ "${SOURCE_FILE_EXT}" == "ova" ]]; then
  echo "Import: Unpacking VMDK files from ova."
  VMDK="$(tar --list -f ${IMAGE_PATH} | grep -m1 vmdk)"
  tar -C /daisy-scratch -xf ${IMAGE_PATH} ${VMDK}
  IMAGE_PATH="/daisy-scratch/${VMDK}"
  echo "Import: New source file is ${VMDK}"
fi

# Ensure the output disk has sufficient space to accept the disk image.
# Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" ${IMAGE_PATH} | grep -m1 "virtual-size" | grep -o '[0-9]\+')
 # Round up to the next GB.
SIZE_GB=$(awk "BEGIN {print int(((${SIZE_BYTES} - 1)/${BYTES_1GB}) + 1)}")
echo "Import: Importing ${IMAGE_PATH} of size ${SIZE_GB}GB to ${DISKNAME} in ${ZONE}." 2> /dev/null

# Ensure the disk referenced by $DISKNAME is large enough to
# hold the inflated disk. For the common case, we initialize
# it to have a capacity of 10 GB, and then resize it if qemu-img
# tells us that it will be larger than 10 GB.
if [[ ${SIZE_GB} -gt 10 ]]; then
  resizeDisk "${DISKNAME}" "${SIZE_GB}" "${ZONE}"
fi

if ! out=$(gcloud -q compute instances attach-disk ${ME} --disk=${DISKNAME} --zone=${ZONE} 2>&1); then
  echo "ImportFailed: Failed to attach disk [Privacy-> from ${DISKNAME} to ${ME}, error: ${out} <-Privacy]"
  exit
fi
echo ${out}

# Convert the image and write it to the disk referenced by $DISKNAME.
# /dev/sdc is used since we're manually attaching this disk, and sdb was already
# used by the scratch disk.
if ! out=$(qemu-img convert ${IMAGE_PATH} -p -O raw -S 512b /dev/sdc 2>&1); then
  echo "ImportFailed: Failed to convert source to raw. [Privacy-> error: ${out} <-Privacy]"
  exit
fi
echo ${out}

sync
gcloud -q compute instances detach-disk ${ME} --disk=${DISKNAME} --zone=${ZONE}

echo "ImportSuccess: Finished import." 2> /dev/null
