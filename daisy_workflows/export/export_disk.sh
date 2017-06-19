#!/bin/bash
# Copyright 2017 Google Inc. All Rights Reserved.
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

echo "Installing necessary packages..."
apt-get update
apt-get -q -y install git-core

function exit_error
{
  echo "export failed"
  exit 1
}

wget --quiet https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz || exit_error
tar -C /usr/local -xzf go1.8.3.linux-amd64.tar.gz || exit_error
export PATH=$PATH:/usr/local/go/bin
export GOPATH=~/go
export PATH=$PATH:~/go/bin
go get -t -v github.com/GoogleCloudPlatform/compute-image-tools/gce_export || exit_error

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/gcs-path)
LICENSES=$(curl -f -H Metadata-Flavor:Google ${URL}/licenses)

echo "Uploading image."
if [[ -n $LICENSES ]]; then
  gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -licenses "$LICENSES" -y || exit_error
else
  gce_export -gcs_path "$GCS_PATH" -disk /dev/sdb -y || exit_error
fi

echo "export success"