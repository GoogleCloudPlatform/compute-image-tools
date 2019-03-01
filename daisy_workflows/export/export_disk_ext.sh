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

URL="http://metadata/computeMetadata/v1/instance/attributes"
GS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
FORMAT=$(curl -f -H Metadata-Flavor:Google ${URL}/format)

# Strip gs://
LOCAL_PATH=${GS_PATH##*//}
# Create dir for output
OUTS_PATH=${LOCAL_PATH%/*}
mkdir -p "/gs/${OUTS_PATH}"

# Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" /dev/sdb | grep -m1 "virtual-size" | grep -o '[0-9]\+')
 # Round up to the next GB.
SIZE_GB=$(awk "BEGIN {print int((${SIZE_BYTES}/1000000000)+ 1)}")

echo "GCEExport: Exporting disk of size ${SIZE_GB}GB and format ${FORMAT}."

if ! out=$(qemu-img convert /dev/sdb "/gs/${LOCAL_PATH}" -p -O $FORMAT 2>&1); then
  echo "ExportFailed: Failed to export disk source to ${GS_PATH} due to qemu-img error: ${out}"
  exit
fi
echo ${out}

if ! out=$(gsutil cp "/gs/${LOCAL_PATH}" "${GS_PATH}" 2>&1); then
  echo "ExportFiled: Failed to copy output image to ${GS_PATH}, error: ${out}"
  exit
fi
echo ${out}

echo "export success"
sync
