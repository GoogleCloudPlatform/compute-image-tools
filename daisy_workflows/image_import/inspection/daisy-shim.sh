#!/usr/bin/env bash
# Copyright 2020 Google Inc. All Rights Reserved.
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

# A Google Compute Engine instance startup-script that:
#  1. Downloads the files specified in the `daisy-sources-path`
#     metadata variable.
#  2. Installs the disk inspection library.
#  3. Runs disk inspection against /dev/sdb
set -eufx -o pipefail

ROOT="/tmp/build-root-$RANDOM"
SOURCE=$(curl "http://metadata.google.internal/computeMetadata/v1/instance/attributes/daisy-sources-path" -H "Metadata-Flavor: Google")
INSPECT_OS=$(curl "http://metadata.google.internal/computeMetadata/v1/instance/attributes/is-inspect-os" -H "Metadata-Flavor: Google")
mkdir -p "$ROOT" && cd "$ROOT"
gsutil cp -R "$SOURCE/*" .
pip3 install ./compute_image_tools_proto ./boot_inspect

if [[ "$INSPECT_OS" == "true" ]]; then
    cmd="boot-inspect --format=daisy --inspect-os /dev/sdb"
else
    cmd="boot-inspect --format=daisy /dev/sdb"
fi

if bash -c "$cmd"; then
  echo "Success:"
else
  echo "Failed:"
fi
