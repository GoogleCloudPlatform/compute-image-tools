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

RET=0

TARGETS=("cli_tools")
for TARGET in "${TARGETS[@]}"; do
  echo "Running tests on ${TARGET}"
  cd /${REPO_PATH}/${TARGET}

  OUT=${ARTIFACTS}/${TARGET}
  mkdir -p ${OUT}

  go test ./... -race -coverprofile=${OUT}/test-report.out -covermode=atomic -v 2>&1 > test.out
  PARTRET=$?
  echo "${TARGET} test returned ${PARTRET}."
  if [ ${PARTRET} -ne 0 ]; then
    RET=${PARTRET}
  fi

  # Output test report.
  cat test.out | go-junit-report > ${ARTIFACTS}/${TARGET//\//_}_report.xml
  rm test.out
done

exit $RET
