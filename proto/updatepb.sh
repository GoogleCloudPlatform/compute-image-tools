#!/bin/bash
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

# This script generates Python and Go code for each proto file in
# in this directory.

set -e

GO_DST=go/pb
PY_DST=py/compute_image_tools_proto

# Change to the proto directory
cd "$(dirname "$(readlink -f "$0")")"

# Ensure protoc is installed
if ! command -v protoc &> /dev/null
then
    echo "protoc not found. To install see: https://developers.google.com/protocol-buffers/docs/reference/go-generated"
    exit
fi

# Install mypy-protobuf; this creates type hints for the
# generated Python code.
pip3 install -U mypy-protobuf

rm -rf $PY_DST $GO_DST && mkdir -p $PY_DST  $GO_DST

echo "Generating go bindings"
protoc -I=. --python_out=$PY_DST --mypy_out=$PY_DST --go_out=$GO_DST *.proto

echo "Don't run flake8 on gnerated Python files."
for f in $PY_DST/*.py; do
  echo "# Don't run flake8 on gnerated Python files." >> $f
  echo "# flake8: noqa" >> $f
done

echo "Done"
