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

DISK_RESIZING_MON=disk_resizing_mon.sh

URL="http://metadata/computeMetadata/v1/instance/attributes"
GS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
FORMAT=$(curl -f -H Metadata-Flavor:Google ${URL}/format)

# Strip gs://
LOCAL_PATH=${GS_PATH##*//}
# Create dir for output
OUTS_PATH=${LOCAL_PATH%/*}
mkdir -p "/gs/${OUTS_PATH}"

# Prepare disk size info.
# 1. Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" /dev/sdb | grep -m1 "virtual-size" | grep -o '[0-9]\+')
# 2. Round up to the next GB.
SIZE_OUTPUT_GB=$(awk "BEGIN {print int((${SIZE_BYTES}/1000000000) + 1)}")
# 3. Add 5GB of additional space to max size to prevent the corner case that output
# file is slightly larger than source disk.
MAX_BUFFER_DISK_SIZE_GB=$(awk "BEGIN {print int(${SIZE_OUTPUT_GB} + 5)}")

# Prepare buffer disk
echo "GCEExport: Initializing buffer disk for qemu-img output..."
sudo arted -a optimal /dev/sdc mklabel gpt
sudo mkfs.ext4 /dev/sdc
sudo mount /dev/sdc "/gs/${OUTS_PATH}"

# Fetch disk size monitor script from GCS1
DISK_RESIZING_MON_GCS_PATH=gs://${OUTS_PATH%/*}/sources/${DISK_RESIZING_MON}
DISK_RESIZING_MON_LOCAL_PATH=/gs/${DISK_RESIZING_MON}
echo "GCEExport: Copying disk size monitor script..."
if ! out=$(gsutil cp "${DISK_RESIZING_MON_PATH}" "/gs/${OUTS}/${DISK_RESIZING_MON}"); then
  echo "ExportFailed: Failed to copy disk size monitor script. Error: ${out}"
  exit
fi
echo ${out}

echo "GCEExport: Running disk size monitor, which can resize disk to at most ${SIZE_DISK_GB}GB depending on actual output image size."
sh /gs/${OUTS}/${DISK_RESIZING_MON} &

echo "GCEExport: Exporting disk of size ${SIZE_OUTPUT_GB}GB and format ${FORMAT}."
if ! out=$(qemu-img convert /dev/sdb "/gs/${IMAGE_OUTPUT_PATH}" -p -O $FORMAT 2>&1); then
  echo "ExportFailed: Failed to export disk source to ${GS_PATH} due to qemu-img error: ${out}"
  exit
fi
echo ${out}

echo "GCEExport: Copying output image to target GCS path..."
if ! out=$(gsutil cp "/gs/${LOCAL_PATH}" "${GS_PATH}" 2>&1); then
  echo "ExportFailed: Failed to copy output image to ${GS_PATH}, error: ${out}"
  exit
fi
echo ${out}

echo "export success"
sync
