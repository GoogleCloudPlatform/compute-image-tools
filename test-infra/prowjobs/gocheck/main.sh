#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

cd /
REPO_PATH=${REPO_NAME}
mkdir -p ${REPO_PATH}
git clone https://github.com/${REPO_OWNER}/${REPO_NAME} ${REPO_PATH}

# Pull PR if this is a PR.
if [ ! -z "${PULL_NUMBER}" ]; then
  cd ${REPO_PATH}
  git fetch origin pull/${PULL_NUMBER}/head:${PULL_NUMBER}
  git checkout ${PULL_NUMBER}
fi

GOFMT_RET=0
GOLINT_RET=0
GOVET_RET=0

TARGETS=("daisy"
         "cli_tools"
         "test-infra/prowjobs/cleanerupper"
         "test-infra/prowjobs/wrapper"
         "cli_tools_e2e_test/gce_image_import_export"
         "cli_tools_e2e_test/gce_ovf_import")
for TARGET in "${TARGETS[@]}"; do
  echo "Checking ${TARGET}"
  cd "/${REPO_PATH}/${TARGET}"
  # We dont run golint on Windows only code as style often matches win32 api 
  # style, not golang style
  golint -set_exit_status ./...
  RET=$?
  if [ $RET != 0 ]; then
    GOLINT_RET=$RET
    echo "'golint ${TARGET}/...' returned ${GOLINT_RET}"
  fi

  GOFMT_OUT=$(gofmt -d $(find . -type f -name "*.go") 2>&1)
  if [ ! -z "${GOFMT_OUT}" ]; then
    echo "'gofmt -d \$(find ${TARGET} -type f -name \"*.go\")' returned:"
    echo ${GOFMT_OUT}
    GOFMT_RET=1
  fi

  go vet --structtag=false ./...
  RET=$?
  if [ $RET != 0 ]; then
    GOVET_RET=$RET
    echo "'go vet ${TARGET}/...' returned ${GOVET_RET}"
  fi
  GOOS=windows go vet --structtag=false ./...
  RET=$?
  if [ $RET != 0 ]; then
    GOVET_RET=$RET
    echo "'GOOS=windows go vet ${TARGET}/...' returned ${GOVET_RET}"
  fi
done

if [ ${GOLINT_RET} != 0 ] || [ ${GOFMT_RET} != 0 ] || [ ${GOVET_RET} != 0 ]; then
  exit 1
fi
exit 0
