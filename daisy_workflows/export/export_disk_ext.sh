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
GCS_PATH=${GS_PATH##*//}
# Grab only the bucket
GCS_BUCKET=${GCS_PATH%%/*}
# Mount GCS bucket containing the disk image.
mkdir -p "/gcs/${GCS_BUCKET}"
gcsfuse ${GCS_BUCKET} /gcs/${GCS_BUCKET}
if [ $? -ne 0 ]; then
  echo "ExportFailed: Failed to mount ${GCS_BUCKET}."
fi
OUTS=${GCS_PATH%/*}
mkdir -p "/gcs/${OUTS}"

# Disk image size info.
SIZE_BYTES=$(qemu-img info --output "json" /dev/sdb | grep -m1 "virtual-size" | grep -o '[0-9]\+')
 # Round up to the next GB.
SIZE_GB=$(awk "BEGIN {print int((${SIZE_BYTES}/1000000000)+ 1)}")

echo "GCEExport: Exporting disk of size ${SIZE_GB}GB and format ${FORMAT}."
qemu-img convert /dev/sdb "/gcs/${GCS_PATH}" -p -O $FORMAT &> qemu_output.txt
QEMU_ERR=$?
if [ ${QEMU_ERR} -ne 0 ]; then
	echo "ExportFailed: Failed to export disk source to ${GS_PATH} due to qemu-img error code '${QEMU_ERR}', error info: $(cat qemu_output.txt)"
fi

echo "export success"
sync
