#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
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

set -e

function exit_error
{
  echo "build failed"
  exit 1
}

trap exit_error ERR

URL="http://metadata/computeMetadata/v1/instance/attributes"
BASE_REPO=$(curl -f -H Metadata-Flavor:Google ${URL}/base-repo)
REPO_BRANCH=$(curl -f -H Metadata-Flavor:Google ${URL}/branch)
TEST_PROJECT_ID=$(curl -f -H Metadata-Flavor:Google ${URL}/test-project-id)
TEST_ZONE=$(curl -f -H Metadata-Flavor:Google ${URL}/test-zone)

# Optional fields
TEST_SUITE_FILTER=$(curl -f -H Metadata-Flavor:Google ${URL}/test-suite-filter) || true
TEST_CASE_FILTER=$(curl -f -H Metadata-Flavor:Google ${URL}/test-case-filter) || true

GOLANG="go1.12.1.linux-amd64.tar.gz"
GO=/tmp/go/bin/go
export GOPATH=/usr/share/gocode
export GOCACHE=/tmp/.cache

apt-get install -y git-core
echo "cloning package"
git clone "https://github.com/${BASE_REPO}/compute-image-tools.git"
cd compute-image-tools
git checkout ${REPO_BRANCH}

# Golang setup
[[ -d /tmp/go ]] && rm -rf /tmp/go
mkdir -p /tmp/go/
echo "Downloading Go"
curl -s "https://dl.google.com/go/${GOLANG}" -o /tmp/go/go.tar.gz
echo "Extracting Go"
tar -C /tmp/go/ --strip-components=1 -xf /tmp/go/go.tar.gz

echo "Pulling dependencies"
sudo su -c "GOPATH=${GOPATH} ${GO} get -d ./..."

CMD="osconfig_tests/main.go"
ARGS="-test_project_id ${TEST_PROJECT_ID} -test_zone ${TEST_ZONE}"

if [ ! -z "$TEST_SUITE_FILTER" ]
then
    ARGS="${ARGS} -test_suite_filter $TEST_SUITE_FILTER"
fi

if [ ! -z "$TEST_CASE_FILTER" ]
then
    ARGS="${ARGS} -test_case_filter $TEST_CASE_FILTER"
fi


echo "running $CMD $ARGS..."
${GO} run ${CMD} ${ARGS}

retVal=$?
if [ $retVal -ne 0 ]; then
    echo "End to End Test run failed"
else
    echo "End to End Test run succeeded"
fi

exit 0