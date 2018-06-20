#!/bin/bash

FAIL=0
FAILURES=""

function status {
  local message="${1}"
  echo "STATUS: ${message}"
}

function fail {
  local message="${1}"
  FAIL=$((FAIL+1))
  FAILURES+="TestFailed: $message"$'\n'
}

# Run tests.
sources_dir=$(curl 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/daisy-sources-path' -H 'Metadata-Flavor: Google')

wget "${sources_dir}/local_file.txt"
if [[ -e 'local_file.txt' ]]; then
  status 'successfully retrieved local file.'
fi

wget "${sources_dir}/gcs_file.txt"
if [[ -e 'gcs_file.txt' ]]; then
  status 'successfully retrieved GCS file.'
fi

# Return results.
if [[ ${FAIL} -eq 0 ]]; then
  echo "PASSED: All tests passed!"
else
  echo "${FAIL} tests failed."
  echo "${FAILURES}"
  echo "FAILED: $0 failed."
fi
