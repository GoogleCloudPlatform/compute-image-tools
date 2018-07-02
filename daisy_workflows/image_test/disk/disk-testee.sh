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

# If first boot, just shutdown the machine and save a file to guarantee it's the
# same disk after resize

if [ ! -e /booted ]; then
    logger -p daemon.info "BOOTED"
    echo "BOOTED" > /booted
    # should power off now, but it's safer to wait for daisy to do that after
    # the BOOTED message was spotted.
else
    # Output the partition table
    parted -l | grep "Partition Table:" | sed -e 's/^/DiskResize: /' | logger -p daemon.info

    # Verify if there is any relevant unallocated disk space
    parted /dev/sda unit GB print free \
      | grep "Free Space" \
      | awk '{ print $3 }' \
      | grep -v "0\.00GB"

    # Output if there is any reasonable free partition
    [ $? -ne 0 ] && \
      logger -p daemon.info "DiskResize: The root partition is occupying the whole disk now" || \
      logger -p daemon.info "DiskResize: There is unallocated space on the disk after growing of root partition"

    # Finish test
    logger -p daemon.info "DiskTestFinished"
fi
