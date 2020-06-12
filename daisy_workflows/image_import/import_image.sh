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

# Enable case-insensitive match; used by fileIsOva.
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
SOURCE_URL="$DAISY_SOURCE_URL/source_disk_file"
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


function resizeDisk() {
  local diskId="${1}"
  local requiredSizeInGb="${2}"
  local zone="${3}"
  local deviceId="${4}"

  echo "Import: Resizing ${diskId} to ${requiredSizeInGb}GB in ${zone}."
  if ! out=$(gcloud -q compute disks resize "${diskId}" --size="${requiredSizeInGb}"GB --zone="${zone}" 2>&1 | tr "\n\r" " "); then
    if echo "$out" | grep -qP "compute\.disks\.resize"; then
      echo "ImportFailed: Failed to resize disk. The Compute Engine default service account needs the role: roles/compute.storageAdmin"
    else
      echo "ImportFailed: Failed to resize disk. [Privacy-> resize disk ${diskId} to ${requiredSizeInGb}GB in ${zone}, error: ${out} <-Privacy]"
    fi
    exit
  fi

  echo "Import: Checking for ${deviceId} ${requiredSizeInGb}G"
  for t in {1..60}; do
    if [[ -e ${deviceId} ]]; then
      local actualSizeBytes=$(lsblk "${deviceId}" --output=size -b | sed -n 2p)
      local actualSizeGb=$(awk "BEGIN {print int(${actualSizeBytes}/${BYTES_1GB})}")
      if [[ "${actualSizeGb}" == "${requiredSizeInGb}" ]]; then
        echo "Import: ${deviceId} is attached and ready."
        return
      fi
    fi
    sleep 5
  done

  echo "ImportFailed: Failed to attach disk ${deviceId}"
  exit
}

# Determines whether a file is an OVA archive; the check isn't
# comprehensive to *all* OVAs, but focuses on the OVAs supported
# by this script (OVAs that use VMDKs).
# Arguments:
#   File to check.
# Returns:
#   0 if the file is a tar file, non-zero otherwise.
function fileIsOvaWithVmdk() {
  if [[ $(file --mime-type "$1") =~ application/x-tar ]]; then
    contents=$(tar --list -f "$1")
    if [[ "$contents" =~ ovf && "$contents" =~ vmdk ]]; then
      return 0
    fi
  fi
  return 1
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
  if fileIsOvaWithVmdk "${IMAGE_PATH}"; then
    scratchDiskSizeGigabytes=$((scratchDiskSizeGigabytes * 2))
  fi

  # This disk is initially created with 10GB of space.
  # Enlarge it if that's insufficient to hold the input image.
  if [[ ${scratchDiskSizeGigabytes} -gt 10 ]]; then
    resizeDisk "${SCRATCH_DISK_NAME}" "${scratchDiskSizeGigabytes}" "${ZONE}" /dev/sdb
  fi



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

function serialOutputKeyValuePair() {
  echo "<serial-output key:'$1' value:'$2'>"
}

copyImageToScratchDisk

# If the image is an OVA, then copy out its VMDK.
if fileIsOvaWithVmdk "${IMAGE_PATH}"; then
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

set +x
echo "Import: $(serialOutputKeyValuePair "target-size-gb" "${SIZE_GB}")"
echo "Import: $(serialOutputKeyValuePair "source-size-gb" "${SOURCE_SIZE_GB}")"
echo "Import: $(serialOutputKeyValuePair "import-file-format" "${IMPORT_FILE_FORMAT}")"
set -x

# Ensure the disk referenced by $DISKNAME is large enough to
# hold the inflated disk. For the common case, we initialize
# it to have a capacity of 10 GB, and then resize it if qemu-img
# tells us that it will be larger than 10 GB.
if [[ ${SIZE_GB} -gt 10 ]]; then
  resizeDisk "${DISKNAME}" "${SIZE_GB}" "${ZONE}" /dev/sdc
fi

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

sync

echo "ImportSuccess: Finished import." 2> /dev/null
