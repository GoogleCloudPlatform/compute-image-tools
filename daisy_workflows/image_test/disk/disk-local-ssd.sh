#!/bin/sh
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

# Test if the local ssd is working or not. Partition it, format it, place some
# file, unmount it and check if it's still there after remount.

# This script runs for testing SCSI and NVMe interfaces.

# Wait for all modules to load. Maybe nvme is not up yet
sleep 10

# Assuming NVMe first
DEVICE=/dev/nvme0n1
PARTITION=/dev/nvme0n1p1
[ -e "$DEVICE" ] && IS_SCSI=0 || IS_SCSI=1

if [ $IS_SCSI -eq 1 ]; then
  DEVICE=/dev/sdb
  PARTITION=/dev/sdb1

  # check for Multiqueue SCSI
  grep scsi_mod.use_blk_mq=Y /proc/cmdline &> /dev/null \
    && logger -p daemon.info "Multiqueue is enable" \
    || logger -p daemon.info "Multiqueue is DISABLED"
fi

MOUNT_POINT=/local_ssd
CHECK_FILENAME=file
CHECK_STRING=DiskWorks

# create a primary partition allocating the whole disk
echo "n




w" | fdisk $DEVICE

mkfs.ext4 $PARTITION
mkdir $MOUNT_POINT
mount $PARTITION $MOUNT_POINT
echo $CHECK_STRING > $MOUNT_POINT/$CHECK_FILENAME
umount $MOUNT_POINT

mount $PARTITION $MOUNT_POINT
if [ $CHECK_STRING = $(cat $MOUNT_POINT/$CHECK_FILENAME) ]; then
  logger -p daemon.info "CheckSuccessful"
else
  logger -p daemon.info "CheckFailed"
fi
