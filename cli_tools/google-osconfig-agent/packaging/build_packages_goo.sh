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

function exit_error
{
  echo "Package build failed"
  exit 1
}

URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_PATH=$(curl -f -H Metadata-Flavor:Google ${URL}/daisy-outs-path)
BASE_REPO=$(curl -f -H Metadata-Flavor:Google ${URL}/base-repo)

apt-get install -y git-core || exit_error
git clone "https://github.com/${BASE_REPO}/compute-image-tools.git" || exit_error
cd compute-image-tools/cli_tools/google-osconfig-agent || exit_error
packaging/setup_goo.sh || exit_error
gsutil cp google-osconfig-agent*.goo "${GCS_PATH}/" || exit_error

echo 'Package build success'
