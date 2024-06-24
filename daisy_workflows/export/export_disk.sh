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
# The gcs root for sbom-util. If empty, do not run sbom generation with sbom-util.
SBOM_UTIL_GCS_ROOT=$(curl -f -H Metadata-Flavor:Google ${URL}/sbom-util-gcs-root)
# Mostly used for windows workflows, set to true if the sbom is already generated and non-empty.
SBOM_ALREADY_GENERATED=$(curl -f -H Metadata-Flavor:Google ${URL}/sbom-already-generated)

# The sha256 sum local text file path used for the sbom: same functionality as $SBOM_PATH.
SHA256_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/sha256-path)

# This function fetches the sbom-util executable from the gcs bucket.
function fetch_sbomutil() {
  echo "GCEExport: provided root path for sbom-util at [${SBOM_UTIL_GCS_ROOT}]"
  if [ -z "${SBOM_UTIL_GCS_ROOT}" ]; then
    echo "GCEExport: SBOM_UTIL_GCS_ROOT is not defined, skipping sbomutil deployment..."
    return
  fi

  SBOM_UTIL_GCS_ROOT="${SBOM_UTIL_GCS_ROOT}/linux"

  # suffix the gcs path with arm64 if necessary
  if [ "$(uname -m)" == "aarch64" ]; then
    SBOM_UTIL_GCS_ROOT="${SBOM_UTIL_GCS_ROOT}_arm64"
  fi

  echo "GCEExport: listing sbom-util versions at [${SBOM_UTIL_GCS_ROOT}]"
  # Determine the latest sbomutil gcs path if available
  if [ -n "${SBOM_UTIL_GCS_ROOT}" ]; then
    SBOM_UTIL_GCS_PATH=$(gsutil ls $SBOM_UTIL_GCS_ROOT | tail -1)
    echo "GCEExport: gsutil list completed with status [$?]"
  fi

  echo "GCEExport: searching for sbom-util at [${SBOM_UTIL_GCS_PATH}]"
  # Fetch sbomutil from gcs if available
  if [ -n "${SBOM_UTIL_GCS_PATH}" ]; then
    echo "GCEExport: Fetching sbomutil: ${SBOM_UTIL_GCS_PATH}"
    gsutil cp "${SBOM_UTIL_GCS_PATH%/}/sbomutil" sbomutil
    chmod +x sbomutil
  fi
}

# This function runs SBOM generation using either syft or sbom-util, depending on the first argument.
function runSBOMGeneration() {
  # Get the partition with the largest size from the mounted disk by sorting.
  SBOM_DISK_PARTITION=$(lsblk $SOURCE_DISK_SYMPATH --output=name -l -b --sort=size | tail -2 | head -1)
  mount /dev/$SBOM_DISK_PARTITION /mnt
  mount -o bind,ro /dev /mnt/dev
  echo "GCEExport: Running sbom generation with the sbom-util program"
  fetch_sbomutil
  ./sbomutil --archetype=linux-image --comp_name=$SOURCE_DISK_NAME --output=image.sbom.json
  gsutil cp image.sbom.json $SBOM_PATH
  umount /mnt/dev
  umount /mnt
  echo "GCEExport: SBOM export success"
}

# This function generates the sha256 hash of the image, used for the sbom.
function generateHash() {
  echo "GCEExport: sha256 text file destination passed in as ${SHA256_PATH}"
  # According to https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/cli_tools/gce_export#compute-engine-image-export,
  # no local image file is stored when gce_export is called. We must copy the image file to calculate the sha256 sum.
  gsutil cp $GCS_PATH img.tar.gz
  if [ $? != 0 ]; then exit 1
    echo "ExportFailed: copying the image tar file locally from ${GCS_PATH} failed"
    exit 1
  fi
  imghash=$(sha256sum ((localfile)).tar.gz | awk '{print $1;}')
  echo "GCEExport: got imghash ${imghash}"
  echo $imghash | tee sha256.txt
  gsutil cp sha256.txt $SHA256_PATH
  rm img.tar.gz
  echo "GCEExport: successfully stored sha256 sum in ${SHA256_PATH}"
}

# Always create empty sbom file so workflow copying does not fail.
if [ $SBOM_ALREADY_GENERATED != "true" ]; then
  touch image.sbom.json
  gsutil cp image.sbom.json $SBOM_PATH
fi
# If the sbom-util program location is passed in, generate the sbom.
if [ $SBOM_UTIL_GCS_ROOT != "" ]; then
  runSBOMGeneration
fi


if [ $SHA256_PATH != "" ]; then
  # Always create the empty sha256 sum text file so workflow copying does not fail.
  touch sha256_sum.txt
  gsutil cp sha256_sum.txt $SHA256_PATH
  generateHash
else
  echo "GCEExport: sha256 text file destination not passed in"
fi

echo "ExportSuccess"
sync
