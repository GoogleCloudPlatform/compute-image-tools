#!/bin/bash
i# Copyright 2019 Google Inc. All Rights Reserved.
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

METADATA_URL="http://metadata/computeMetadata/v1/instance"
SIZE=$(curl -f -H Metadata-Flavor:Google ${METADATA_URL}/attributes/disk-size)
RESERVED_SPACE=50 #50MB reserved space so system can still run

echo "GCECreateDisk: Fullfilling disk of size ${SIZE}GB."
SIZE_MB=$(awk "BEGIN {print int(${SIZE} * 1024)}")
mkfs.ext4 /dev/sdb
mkdir -p /mnt/sdb
mount /dev/sdb /mnt/sdb
dd if=/dev/zero of=/mnt/sdb/reserved.space count=${RESERVED_SPACE} bs=1M
dd if=/dev/urandom of=/mnt/sdb/random.file count=${SIZE_MB} bs=1M
rm /mnt/sdb/reserved.space

echo "create disk success"
sync
