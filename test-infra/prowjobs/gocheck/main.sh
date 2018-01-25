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

set -e
set -x

git clone https://github.com/${REPO_OWNER}/${REPO_NAME} /repo
cd /repo

# Pull PR if this is a PR.
if [ ! -z "${PULL_NUMBER}" ]; then
  git fetch origin pull/${PULL_NUMBER}/head:${PULL_NUMBER}
  git checkout ${PULL_NUMBER}
fi

set +e

golint -set_exit_status ./...
GOLINT_RET=$?
GOFMT_OUT=$(gofmt -l $(find . -type f -name "*.go") 2>&1)
if [ -z "${GOFMT_OUT}" ]; then
  GOFMT_RET=0
else
  GOFMT_RET=1
fi
go vet ./...
GOVET_RET=$?

set +x

# Flush set -x output.
sync

# Print results and return.
if [ ${GOLINT_RET} != 0 ]; then
  echo "'golint ./...' returned ${GOLINT_RET}"
fi
if [ ${GOFMT_RET} != 0 ]; then
  echo "'gofmt -l' returned ${GOFMT_RET}"
fi
if [ ${GOVET_RET} != 0 ]; then
  echo "'go vet ./...' returned ${GOVET_RET}"
fi
if [ ${GOLINT_RET} != 0 ] || [ ${GOFMT_RET} != 0 ] || [ ${GOVET_RET} != 0 ]; then
  exit 1
fi
exit 0
