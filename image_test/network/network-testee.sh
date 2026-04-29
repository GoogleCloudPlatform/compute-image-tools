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

# Signalize wait-for-instance that instance is ready
logger -p daemon.info "BOOTED"

# Serve a file server that prints the hostname when requesting "/hostname"
# and "linux" when requesting "/os"
mkdir /server
cd /server
hostname > hostname
echo linux > os
# host it on python2 and fallback to python3
python -m SimpleHTTPServer 80
python3 -m http.server 80
