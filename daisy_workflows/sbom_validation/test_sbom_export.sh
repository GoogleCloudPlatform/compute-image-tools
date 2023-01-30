#!/bin/bash

URL="http://metadata/computeMetadata/v1/instance/attributes"
OUTS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/outs-path)
DISK_FILE_NAME=$(curl -f -H Metadata-Flavor:Google ${URL}/disk-file-name)

GCS_PATH_SBOM=${OUTS_PATH}/*.sbom.json
GCS_PATH_OUTDISK=${OUTS_PATH}/${DISK_FILE_NAME}
# check if tar.gz file is there
# check if sbom file is there
# potentially run other script for sbom contents
# to check, need a convention for the sbom json file name as well, input variable. 

gsutil -q stat $GCS_PATH_SBOM
status=$?

if [[ $status == 0 ]]; then
  echo "SBOM file successfully found"
else
  echo "SBOMFailed: sbom file not found"
  exit 1
fi

gsutil -q stat $GCS_PATH_OUTDISK
status=$?

if [[ $status == 0 ]]; then
  echo "Disk tar file successfully found"
else
  echo "SBOMFailed: disk tar gz failure"
  exit 1
fi

echo "SBOMTesting: All tests passed"
echo "SBOMSuccess"
sync


