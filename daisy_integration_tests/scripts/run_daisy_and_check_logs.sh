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
INSTANCE_ID="$(curl -f -H Metadata-Flavor:Google ${URL}/id)"
SHOULD_HAVE_LOGS="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/should_have_logs)"

# Install build dependencies.
apt-get update
apt-get -y install git
if [ $? -ne 0 ]; then
  echo "BuildFailed: Unable to install build dependencies."
  exit 1
fi

wget --quiet https://dl.google.com/go/go1.10.3.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.10.3.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Pull daisy
go get -v github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisy
if [ $? -ne 0 ]; then
  echo "BuildFailed: Error pulling Daisy."
  exit 1
fi

# Run daisy
daisy go/src/github.com/GoogleCloudPlatform/compute-image-tools/daisy_integration_tests/step_create_disks.wf.json
if [ $? -ne 0 ]; then
  echo "BuildFailed: Error executing Daisy."
  exit 1
fi

# Verify logs were sent to Cloud Logging
LOGS=$(gcloud logging read "resource.labels.instance_id=$INSTANCE_ID jsonPayload.workflow=create-disks-test")
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
