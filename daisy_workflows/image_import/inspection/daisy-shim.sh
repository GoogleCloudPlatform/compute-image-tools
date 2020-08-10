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
#  2. Executes the file `inspect-disk`.
set -euf -o pipefail

ROOT="/tmp/build-root-$RANDOM"
SOURCE=$(curl "http://metadata.google.internal/computeMetadata/v1/instance/attributes/daisy-sources-path" -H "Metadata-Flavor: Google")

mkdir -p "$ROOT" && cd "$ROOT"
gsutil cp -R "$SOURCE/*" .
python3 inspect-disk.py --format=daisy /dev/sdb
echo "Success:"
