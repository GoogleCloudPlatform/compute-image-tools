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

function serialOutputPrefixedKeyValue() {
  stdbuf -oL echo "$1: <serial-output key:'$2' value:'$3'>"
}

# Verify VM has access to Google APIs
curl --silent --fail "https://www.googleapis.com/discovery/v1/apis" &> /dev/null;
if [[ $? -ne 0 ]]; then
  echo "ExportFailed: Cannot access Google APIs. Ensure that VPC settings allow VMs to access Google APIs either via external IP or Private Google Access. More info at: https://cloud.google.com/vpc/docs/configure-private-google-access"
  exit
fi

BYTES_1GB=1073741824
URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
LICENSES=$(curl -f -H Metadata-Flavor:Google ${URL}/licenses)
# Name for the source disk when getting the sympath
SOURCE_DISK_NAME=$(curl -f -H Metadata-Flavor:Google ${URL}/source-disk-name)

mkdir ~/upload

# Source disk size info.
SOURCE_DISK_SYMPATH=$(readlink -f /dev/disk/by-id/google-$SOURCE_DISK_NAME)
SOURCE_SIZE_BYTES=$(lsblk $SOURCE_DISK_SYMPATH --output=size -b | sed -n 2p)
SOURCE_SIZE_GB=$(awk "BEGIN {print int(((${SOURCE_SIZE_BYTES}-1)/${BYTES_1GB}) + 1)}")
serialOutputPrefixedKeyValue "GCEExport" "source-size-gb" "${SOURCE_SIZE_GB}"

echo "GCEExport: Running export tool."
if [[ -n $LICENSES ]]; then
  gce_export -buffer_prefix ~/upload -gcs_path "$GCS_PATH" -disk $SOURCE_DISK_SYMPATH -licenses "$LICENSES" -y
else
  gce_export -buffer_prefix ~/upload -gcs_path "$GCS_PATH" -disk $SOURCE_DISK_SYMPATH -y
fi
GCE_EXPORT_RESULT=$?
if [[ ${GCE_EXPORT_RESULT} -eq 2 ]]; then
  echo "ExportFailed: Failed to export disk source to GCS because default Compute Engine Service account does not have storage.objects.create permission for [Privacy-> ${GCS_PATH}. <-Privacy]."
  exit 1
fi
if [[ ${GCE_EXPORT_RESULT} -ne 0 ]]; then
  echo "ExportFailed: Failed to export disk source to GCS [Privacy-> ${GCS_PATH}. <-Privacy]."
  exit 1
fi

# Exported image size info.
TARGET_SIZE_BYTES=$(gsutil ls -l "${GCS_PATH}" | head -n 1 | awk '{print $1}')
TARGET_SIZE_GB=$(awk "BEGIN {print int(((${TARGET_SIZE_BYTES}-1)/${BYTES_1GB}) + 1)}")
serialOutputPrefixedKeyValue "GCEExport" "target-size-gb" "${TARGET_SIZE_GB}"
# Final destination for the disk and sbom exported files
DESTINATION=$(curl -f -H Metadata-Flavor:Google ${URL}/destination)
# Local sbom outspath
SBOM_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/sbom-path)
# References the tar-gz for syft, if SBOM generation will run. 
SYFT_TAR_FILE=$(curl -f -H Metadata-Flavor:Google ${URL}/syft-tar-file)
# User passed in value for syft source, if empty then do not run SBOM generation
SYFT_SOURCE=$(curl -f -H Metadata-Flavor:Google ${URL}/syft-source)

function runSBOMGeneration() {
  # Get the partition with the largest size from the mounted disk by sorting
  SBOM_DISK_PARTITION=$(lsblk $SOURCE_DISK_SYMPATH --output=name -l -b --sort=size | tail -2 | head -1)
  mount /dev/$SBOM_DISK_PARTITION /mnt
  mount -o bind,ro /dev /mnt/dev
  gsutil cp $SYFT_TAR_FILE syft.tar.gz
  tar -xf syft.tar.gz
  ./syft /mnt -o spdx-json > image.sbom.json
  gsutil cp image.sbom.json $SBOM_PATH
  gsutil cp $SBOM_PATH $DESTINATION
  umount /mnt/dev
  umount /mnt
  echo "GCEExport: SBOM export success"
}

# If no source for syft was passed in, do not run SBOM generation. 
if [ $SYFT_SOURCE != "" ]; then
  runSBOMGeneration
fi

echo "ExportSuccess"
sync
