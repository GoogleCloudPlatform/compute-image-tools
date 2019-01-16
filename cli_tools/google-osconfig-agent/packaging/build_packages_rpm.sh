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
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/daisy-outs-path)
BASE_REPO=$(curl -f -H Metadata-Flavor:Google ${URL}/base-repo)

n=0
while ! yum install -y git-core; do
  if [[ n -gt 3 ]]; then
    exit 1
  fi
  n=$[$n+1]
  sleep 5
done

git clone "https://github.com/${BASE_REPO}/compute-image-tools.git"
cd compute-image-tools/cli_tools/google-osconfig-agent 
packaging/setup_rpm.sh
gsutil cp /tmp/rpmpackage/RPMS/x86_64/google-osconfig-agent-*.rpm "${GCS_PATH}/"

echo 'Package build success'
