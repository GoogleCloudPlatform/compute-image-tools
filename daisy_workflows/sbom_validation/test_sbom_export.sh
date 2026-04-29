#!/bin/bash

URL="http://metadata/computeMetadata/v1/instance/attributes"
OUTS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/outs-path)
# If set to false, do not check if the disk tar file is present
CHECK_DISK_TAR=$(curl -f -H Metadata-Flavor:Google ${URL}/check-disk-tar)
DISK_FILE_NAME=$(curl -f -H Metadata-Flavor:Google ${URL}/disk-file-name)

# expected GCS path for the sbom file ending in .sbom.json
GCS_PATH_SBOM=${OUTS_PATH}/*.sbom.json
GCS_PATH_OUTDISK=${OUTS_PATH}/${DISK_FILE_NAME}

# Ensure that there are not multiple sbom files
gsutil du $GCS_PATH_SBOM > sbom_file_info.txt
num_sbom_files=$(wc -l < sbom_file_info.txt)
if [[ $num_sbom_files -eq 1 ]]; then
  echo "SBOMTesting: found exactly one SBOM file"
else
  echo "SBOMFailed: incorrect number of SBOM files found"
fi

# The disk export workflow creates an empty sbom file by default
# so check that it has been overriden with a non-empty sbom
read -r -n 1 sbom_file_size < sbom_file_info.txt
if [[ $sbom_file_size -eq 0 ]]; then
  echo "SBOMFailed: empty SBOM file found"
else
  echo "SBOMTesting: non-empty SBOM file found"
fi

# Check that the disk export succeeded, if the flag is not false
if [[ $CHECK_DISK_TAR == "false" ]]; then
  echo "SBOMTesting: skipping check for Disk tar file"
else
  gsutil -q stat $GCS_PATH_OUTDISK
  status=$?
  if [[ $status -eq 0 ]]; then
    echo "SBOMTesting: Disk tar file successfully found"
  else
    echo "SBOMFailed: Disk tar file not found"
    exit 1
  fi
fi

echo "SBOMTesting: All tests passed"
echo "SBOMSuccess"
sync
