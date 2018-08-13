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

URL="http://metadata/computeMetadata/v1/instance"
DAISY_SOURCE_URL="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/daisy-sources-path)"
SOURCEURL="${DAISY_SOURCE_URL}/source_disk_file"
SOURCEBUCKET="$(echo ${SOURCEURL} | awk -F/ '{print $3}')"
SOURCEPATH="${SOURCEURL#"gs://"}"
SOURCE_DISK_FILE="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/source_disk_file)"
DISKNAME="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/disk_name)"
ME="$(curl -f -H Metadata-Flavor:Google ${URL}/name)"
ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/zone)

# Print info.
echo "#################" 2> /dev/null
echo "# Configuration #" 2> /dev/null
echo "#################" 2> /dev/null
echo "SOURCEURL: ${SOURCEURL}" 2> /dev/null
echo "SOURCEBUCKET: ${SOURCEBUCKET}" 2> /dev/null
echo "SOURCEPATH: ${SOURCEPATH}" 2> /dev/null
echo "DISKNAME: ${DISKNAME}" 2> /dev/null
echo "ME: ${ME}" 2> /dev/null
echo "ZONE: ${ZONE}" 2> /dev/null

# Mount GCS bucket containing the disk image.
mkdir -p /gcs/${SOURCEBUCKET}
gcsfuse --implicit-dirs ${SOURCEBUCKET} /gcs/${SOURCEBUCKET}

# Atrocious OVA hack.
SOURCEFILE_TYPE="${$SOURCE_DISK_FILE##*.}"
if [[ "${SOURCEFILE_TYPE}" == "ova" ]]; then
  echo "Import: Unpacking VMDK files from ova."
  VMDK="$(tar --list -f /gcs/${SOURCEPATH} | grep -m1 vmdk)"
  tar -C /gcs/${DAISY_SOURCE_URL#"gs://"} -xf /gcs/${SOURCEPATH} ${VMDK}
  SOURCEPATH="${DAISY_SOURCE_URL#"gs://"}/${VMDK}"
  echo "Import: New source file is ${VMDK}"
fi

# Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" /gcs/${SOURCEPATH} | grep -m1 "virtual-size" | grep -o '[0-9]\+')
 # Round up to the next GB.
SIZE_GB=$(awk "BEGIN {print int((${SIZE_BYTES}/1000000000)+ 1)}")

echo "Import: Importing ${SOURCEPATH} of size ${SIZE_GB}GB to ${DISKNAME} in ${ZONE}." 2> /dev/null

# Resize the disk if its bigger than 10GB and attach it.
if [[ ${SIZE_GB} -gt 10 ]]; then
  gcloud -q compute disks resize ${DISKNAME} --size=${SIZE_GB}GB --zone=${ZONE}
  if [ $? -ne 0 ]; then
    echo "ImportFailed: Failed to resize ${DISKNAME} to ${SIZE_GB}GB in ${ZONE}" 2> /dev/null
  fi
fi

gcloud -q compute instances attach-disk ${ME} --disk=${DISKNAME} --zone=${ZONE}
if [ $? -ne 0 ]; then
  echo "ImportFailed: Failed to attach ${DISKNAME} to ${ME}" 2> /dev/null
fi

# Write imported disk to GCE disk.
qemu-img convert /gcs/${SOURCEPATH} -p -O raw -S 512b /dev/sdb
if [ $? -ne 0 ]; then
  echo "ImportFailed: Failed to convert source to raw." 2> /dev/null
fi

sync
gcloud -q compute instances detach-disk ${ME} --disk=${DISKNAME} --zone=${ZONE}

echo "ImportSuccess: Finished import." 2> /dev/null
