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

wget --quiet https://storage.googleapis.com/compute-image-tools/release/linux/gce_export
if [ $? -ne 0 ]; then
  echo "ExportFailed: Unable to download gce_export."
fi
chmod +x ./gce_export

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
LICENSES=$(curl -f -H Metadata-Flavor:Google ${URL}/licenses)

echo "Uploading image."
if [[ -n $LICENSES ]]; then
  ./gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -licenses "$LICENSES" -y || exit_error
else
  ./gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -y || exit_error
fi
if [ $? -ne 0 ]; then
  echo "ExportFailed: Unable to install gcsfuse or qemu-utils."
fi

echo "export success"
