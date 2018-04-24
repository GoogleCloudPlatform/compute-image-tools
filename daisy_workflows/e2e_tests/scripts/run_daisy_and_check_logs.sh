#!/bin/bash
# Copyright 2018 Google Inc. All Rights Reserved.
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

URL="http://metadata/computeMetadata/v1/instance"
BRANCH="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/github_branch)"
GIT_REPO="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/github_repo)"
INSTANCE_ID="$(curl -f -H Metadata-Flavor:Google ${URL}/id)"
SHOULD_HAVE_LOGS="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/should_have_logs)"

# Install build dependencies.
apt-get update
apt-get -y install git golang
if [ $? -ne 0 ]; then
  echo "BuildFailed: Unable to install build dependencies."
  exit 1
fi

# Setup GOPATH
mkdir -p go/src/github.com/GoogleCloudPlatform
export GOPATH=$HOME/go

# Clone the github repo.
pushd $GOPATH/src/github.com/GoogleCloudPlatform
git clone ${GIT_REPO} -b ${BRANCH}
if [ $? -ne 0 ]; then
  echo "BuildFailed: Unable to clone github repo ${GIT_REPO} and branch ${BRANCH}"
  exit 1
fi
popd

# Build daisy
pushd $GOPATH/src/github.com/GoogleCloudPlatform/compute-image-tools/daisy
go get ./...
popd

pushd $GOPATH/src/github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisy
go build
popd

# Run daisy
pushd $GOPATH/src/github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisy

./daisy ../../daisy_workflows/e2e_tests/attach_disks.wf.json
if [ $? -ne 0 ]; then
  echo "BuildFailed: Error executing Daisy."
  exit 1
fi
popd

# Verify logs were sent to Cloud Logging
LOGS=$(gcloud logging read "resource.labels.instance_id=$INSTANCE_ID jsonPayload.workflow=attach-disks-test")
if [ $SHOULD_HAVE_LOGS = "true" ]; then
  if [ -z "$LOGS" ]; then
    echo "BuildFailed: Expected Cloud Logs."
    exit 1
  else
    echo "Pass: Found expected Cloud Logs."
  fi
else
  if [ -n "$LOGS" ]; then
    echo "BuildFailed: Expected no Cloud Logs."
    exit 1
  else
    echo "Pass: Cloud Logs were not written."
  fi
fi

echo "BuildSuccess: Daisy completed."
