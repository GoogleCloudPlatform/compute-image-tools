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

# Make sure docker is running
systemctl start docker
sleep 5

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
LICENSES=$(curl -f -H Metadata-Flavor:Google ${URL}/licenses)

echo "GCEExport: Running export tool." > /dev/ttyS0
if [[ -n $LICENSES ]]; then
  docker run -t --privileged gcr.io/compute-image-tools/gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -licenses "$LICENSES" -y > /dev/ttyS0 2>&1
else
  docker run -t --privileged gcr.io/compute-image-tools/gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -y > /dev/ttyS0 2>&1
fi
if [ $? -ne 0 ]; then
  echo "ExportFailed: Failed to export disk source to ${GCS_PATH}." > /dev/ttyS0
  exit 1
fi

echo "ExportSuccess" > /dev/ttyS0
sync
