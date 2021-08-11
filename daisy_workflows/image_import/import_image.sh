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

set -o pipefail

# Enable case-insensitive match when checking for
# the ova filetype.
shopt -s nocasematch

# Verify VM has access to Google APIs
curl --silent --fail "https://www.googleapis.com/discovery/v1/apis" &> /dev/null;
if [[ $? -ne 0 ]]; then
  echo "ImportFailed: Cannot access Google APIs. Ensure that VPC settings allow VMs to access Google APIs either via external IP or Private Google Access. More info at: https://cloud.google.com/vpc/docs/configure-private-google-access"
  exit
fi

BYTES_1GB=1073741824
URL="http://metadata/computeMetadata/v1/instance"
DAISY_SOURCE_URL="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/daisy-sources-path)"
SOURCE_URL="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/source_disk_file)"
DISKNAME="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/disk_name)"
SCRATCH_DISK_NAME="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/scratch_disk_name)"
ME="$(curl -f -H Metadata-Flavor:Google ${URL}/name)"
ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/zone)

SOURCE_SIZE_BYTES="$(gsutil du "${SOURCE_URL}" | grep -o '^[0-9]\+')"
SOURCE_SIZE_GB=$(awk "BEGIN {print int(((${SOURCE_SIZE_BYTES}-1)/${BYTES_1GB}) + 1)}")
IMAGE_PATH="/daisy-scratch/$(basename "${SOURCE_URL}")"


# Print info.
echo "#################" 2> /dev/null
echo "# Configuration #" 2> /dev/null
echo "#################" 2> /dev/null
echo "IMAGE_PATH: ${IMAGE_PATH}" 2> /dev/null
echo "SOURCE_URL: ${SOURCE_URL}" 2> /dev/null
echo "SOURCE_SIZE_BYTES: ${SOURCE_SIZE_BYTES}" 2> /dev/null
echo "DISKNAME: ${DISKNAME}" 2> /dev/null
echo "ME: ${ME}" 2> /dev/null
echo "ZONE: ${ZONE}" 2> /dev/null

# Verifies that device at $devicePath is attached, and has capacity of at least $requiredGb gigabytes.
#
# Positional Args:
#   $devicePath of device to check, eg: /dev/sdb
#   $requiredGb, eg: 12
#
# Returns:
#   0 when device is attached and has sufficient capacity
#  >0 otherwise
function deviceHasCapacity() {
  local devicePath="${1}"
  local requiredGb="${2}"

  printf "ensuring %s has capacity of at least %s GB\n" "$devicePath" "$requiredGb"
  if [[ -e ${devicePath} ]]; then
    lsblk_out=$(lsblk "${devicePath}" --output=size -b)
    printf "lsblk output:\n%s\n" "$lsblk_out"
    local actualBytes=$(echo "$lsblk_out" | sed -n 2p)
    local actualGb=$(awk "BEGIN {print int(${actualBytes}/${BYTES_1GB})}")
    if [[ $actualGb -ge $requiredGb ]]; then
      return
    fi
    printf "actualGb=%s requiredGb=%s\n" "$actualGb" "$requiredGb"
    return 1
  fi

  printf "device %s not found\n" "$devicePath"
  return 1
}

# Verifies that a GCE disk is attached, and that the capacity is at least $requiredGb.
# If insufficient size, gcloud is used to resize the disk.
#
# Positional Args:
#   $diskId to verify, eg: disk-12
#   $requiredGb, eg: 12
#   $zone, eg: us-west1-a
#   $devicePath, eg: /dev/sdb
#
# Returns:
#   0 when device is attached and has sufficient capacity
#   exits program otherwise
function ensureCapacityOfDisk() {
  local diskId="${1}"
  local requiredGb="${2}"
  local zone="${3}"
  local devicePath="${4}"

  echo "Import: Ensuring ${diskId} has capacity of ${requiredGb} GB in ${zone}."
  if deviceHasCapacity "$devicePath" "$requiredGb"; then
    echo "Import: ${devicePath} is attached and ready."
    return
  fi

  if ! out=$(gcloud -q compute disks resize "${diskId}" --size="${requiredGb}"GB --zone="${zone}" 2>&1 | tr "\n\r" " "); then
    if echo "$out" | grep -qiP "compute\.disks\.resize.*permission"; then
      echo $out
      echo "ImportFailed: Failed to resize disk. The Compute Engine default service account needs the role: roles/compute.storageAdmin"
    else
      echo "ImportFailed: Failed to resize disk. [Privacy-> resize disk ${diskId} to ${requiredGb}GB in ${zone}, error: ${out} <-Privacy]"
    fi
    exit
  fi
  echo "Resizing result: $out"

  echo "Import: Checking for ${devicePath} ${requiredGb}G"
  for t in {1..60}; do
    if deviceHasCapacity "$devicePath" "$requiredGb"; then
      echo "Import: ${devicePath} is attached and ready."
      return
    fi
    sleep 5
  done

  echo "ImportFailed: Failed to attach disk ${devicePath}"
  exit
}

function copyImageToScratchDisk() {
  # We allocate an extra 10% capacity to account for ext4's
  # filesystem overhead. According to https://petermolnar.net/why-use-btrfs-for-media-storage/ ,
  # the overhead is less than 10% using the default number of inodes
  # and a reserved block percentage of 1%. We conserve a little more space
  # by avoiding the reserved blocks via `mkfs.ext4 -m 0`.
  local scratchDiskSizeGigabytes=$(awk "BEGIN {print int((${SOURCE_SIZE_GB} * 1.1) + 1)}")
  # We allocate double capacity for OVA, which would
  # require making an additional copy of its enclosed VMDK.
  if [[ "${IMAGE_PATH}" =~ \.ova$ ]]; then
     scratchDiskSizeGigabytes=$((scratchDiskSizeGigabytes * 2))
  fi

  ensureCapacityOfDisk "${SCRATCH_DISK_NAME}" "${scratchDiskSizeGigabytes}" "${ZONE}" /dev/sdb

  mkdir -p /daisy-scratch
  # /dev/sdb is used since the scratch disk is the second
  # disk that's attached in inflate_file.wf.json.
  #
  # We disable reserved blocks to save disk space via `-m 0`. Typically
  # this is 5% and we won't be using it.
  mkfs.ext4 /dev/sdb -m 0
  mount /dev/sdb /daisy-scratch
  if [[ $? -ne 0 ]]; then
    echo "ImportFailed: Failed to prepare scratch disk."
  fi

  # Standard error for `gsutil cp` contains a progress meter that when written
  # to the console will exceed the logging daemon's buffer for large files.
  # The stream may contain useful debugging messages, however, so if there's an
  # error we print any lines that don't have ascii control characters, which
  # are used to generate the progress meter.
  if ! out=$(gsutil cp "${SOURCE_URL}" "${IMAGE_PATH}" 2> gsutil.cp.err); then
    echo "Import: Failure while executing gsutil cp:"
    grep -v '[[:cntrl:]]' gsutil.cp.err | while read line; do
      echo "Import: ${line}"
    done
    if grep -qP "storage\.objects\.(list|get)" gsutil.cp.err; then
      echo "ImportFailed: Failed to download image to worker instance. The Compute Engine default service account needs the role: roles/storage.objectViewer"
    else
      echo "ImportFailed: Failed to download image to worker instance [Privacy-> from ${SOURCE_URL} to ${IMAGE_PATH} <-Privacy]."
    fi
    exit
  fi
  echo "Import: Copied image from ${SOURCE_URL} to ${IMAGE_PATH}: ${out}"
}

function serialOutputPrefixedKeyValue() {
  stdbuf -oL echo "$1: <serial-output key:'$2' value:'$3'>"
}

# Dup logic in api_inflater.go. If change anything here, please change in both places.
function diskChecksum() {
  CHECK_DEVICE=sdc
  BLOCK_COUNT=$(cat /sys/class/block/$CHECK_DEVICE/size)

  # Check size = 200000*512 = 100MB
  CHECK_COUNT=200000
  CHECKSUM1=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=0 count=$CHECK_COUNT | md5sum)
  CHECKSUM2=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( 2000000 - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
  CHECKSUM3=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( 20000000 - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
  CHECKSUM4=$(sudo dd if=/dev/$CHECK_DEVICE ibs=512 skip=$(( $BLOCK_COUNT - $CHECK_COUNT )) count=$CHECK_COUNT | md5sum)
  serialOutputPrefixedKeyValue "Import" "disk-checksum" "$CHECKSUM1-$CHECKSUM2-$CHECKSUM3-$CHECKSUM4"
  for i in {1..5000}; do
    serialOutputPrefixedKeyValue "Import" "disk-checksum" "~~~$i~~~ $CHECKSUM1-$CHECKSUM2-$CHECKSUM3-$CHECKSUM4"
  done
}

copyImageToScratchDisk

# If the image is an OVA, then copy out its VMDK.
if [[ "${IMAGE_PATH}" =~ \.ova$ ]]; then
  echo "Import: Unpacking VMDK files from ova."
  VMDK="$(tar --list -f "${IMAGE_PATH}" | grep -m1 vmdk)"
  tar -C /daisy-scratch -xf "${IMAGE_PATH}" "${VMDK}"
  IMAGE_PATH="/daisy-scratch/${VMDK}"
  echo "Import: New source file is ${VMDK}"
fi

# Check whether the image is valid, and attempt to repair if not.
# We skip repair if the return code is zero (the image is valid) or
# if the return code is 63 (the image doesn't support checks).
#
# The magic number 63 can be found in the man pages for qemu-img, eg:
# https://manpages.debian.org/testing/qemu-utils/qemu-img.1.en.html
qemu-img check "${IMAGE_PATH}"
if ! [[ $? == 0 || $? == 63 ]]; then
  if ! out=$(qemu-img check -r all "${IMAGE_PATH}" 2>&1); then
    echo "ImportFailed: The image file is not decodable. Details: $out"
    exit
  fi
fi

# Ensure the output disk has sufficient space to accept the disk image.
# Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" "${IMAGE_PATH}" | grep -m1 "virtual-size" | grep -o '[0-9]\+')
IMPORT_FILE_FORMAT=$(qemu-img info "${IMAGE_PATH}" | grep -m1 "file format" | grep -oP '(?<=^file format:[ *]).*')
 # Round up to the next GB.
SIZE_GB=$(awk "BEGIN {print int(((${SIZE_BYTES} - 1)/${BYTES_1GB}) + 1)}")
echo "Import: Importing ${IMAGE_PATH} of size ${SIZE_GB}GB to ${DISKNAME} in ${ZONE}." 2> /dev/null

serialOutputPrefixedKeyValue "Import" "target-size-gb" "${SIZE_GB}"
serialOutputPrefixedKeyValue "Import" "source-size-gb" "${SOURCE_SIZE_GB}"
serialOutputPrefixedKeyValue "Import" "import-file-format" "${IMPORT_FILE_FORMAT}"

ensureCapacityOfDisk "${DISKNAME}" "${SIZE_GB}" "${ZONE}" /dev/sdc

# Convert the image and write it to the disk referenced by $DISKNAME.
# /dev/sdc is used since it's the third disk that's attached in inflate_file.wf.json.
if ! out=$(qemu-img convert "${IMAGE_PATH}" -p -O raw -S 512b /dev/sdc 2>&1); then
  if [[ "${IMAGE_PATH}" =~ \.vmdk$ ]]; then
    if file "${IMAGE_PATH}" | grep -qiP ascii; then
      hint="When importing a VMDK disk image, ensure that you specify the VMDK disk "
      hint+="image file, rather than its text descriptor file. In some virtual "
      hint+="machine managers, given a text descriptor called <disk.vmdk>, "
      hint+="the disk image file would be called <disk-flat.vmdk>."
    else
      hint="When preparing a VM import, ensure that you create a monolithic "
      hint+="image file, where the disk is contained in a single VMDK file, as "
      hint+="opposed to a split file, where the disk is spread across multiple files."
    fi
  fi
  echo "Import: [Privacy-> error: ${out} <-Privacy] "
  echo "ImportFailed: Failed to decode image file. $hint"
  exit
fi
echo "${out}"

diskChecksum

sync

echo "ImportSuccess: Finished import." 2> /dev/null
