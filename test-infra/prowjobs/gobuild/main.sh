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

GOBUILD_OUT=0
LINUX_TARGETS=("daisy/cli"
               "cli_tools/gce_export"
               "cli_tools/gce_image_publish" 
               "cli_tools/import_precheck"
               "daisy/daisy_test_runner"
               "cli_tools_e2e_test/gce_image_import_export"
               "cli_tools_e2e_test/gce_ovf_import")
for TARGET in "${LINUX_TARGETS[@]}"; do
  echo "Building ${TARGET} for Linux"
  cd /${REPO_PATH}/${TARGET}
  go build
  RET=$?
  if [ $RET != 0 ]; then
    GOBUILD_OUT=$RET
    echo "'go build' exited with ${GOBUILD_OUT}"
  fi
done

WINDOWS_TARGETS=("daisy/cli"
                 "cli_tools/gce_export"
                 "cli_tools/import_precheck")
for TARGET in "${WINDOWS_TARGETS[@]}"; do
  echo "Building ${TARGET} for Linux"
  cd /${REPO_PATH}/${TARGET}
  echo "Building ${TARGET} for Windows"
  GOOS=windows go build
  RET=$?
  if [ $RET != 0 ]; then
    GOBUILD_OUT=$RET
    echo "'GOOS=windows go build' exited with ${GOBUILD_OUT}"
  fi
done

sync
exit $GOBUILD_OUT

