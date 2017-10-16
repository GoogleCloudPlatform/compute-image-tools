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

set -x

# Download translate_debian.py
URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_SOURCE_PATH="$(curl -f -H Metadata-Flavor:Google ${URL}/daisy-sources-path)"
gsutil cp "${GCS_SOURCE_PATH}/translate_debian.py" /tmp/translate_debian.py

# Install dependencies
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get -y install python-guestfs python-pycurl

# Customize image
python /tmp/translate_debian.py
