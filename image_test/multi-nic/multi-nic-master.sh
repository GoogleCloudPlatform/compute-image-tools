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

# Wait for network initialization
sleep 10

# Ubuntu minimal images do not have ping installed by default.
if [  -n "$(uname -a | grep Ubuntu)" ]; then
    if ! type ping > /dev/null; then
        apt-get update && apt-get install -y iputils-ping
    fi
fi

ping -c 5 ${HOST} && logger -p daemon.info 'MultiNICSuccess' || logger -p daemon.info 'MultiNICFailed'
