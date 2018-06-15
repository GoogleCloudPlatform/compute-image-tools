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

# Verify DNS connections if there is a INSTANCE variable defined

# Verify VM to VM DNS connection
[ -n "$INSTANCE" ] && getent hosts $INSTANCE

# Verify VM to external DNS connection
[ -n "$INSTANCE" ] && getent hosts www.google.com

# Signalize wait-for-instance that instance is ready or error occurred
[ $? -ne 0 ] && [ -n "$INSTANCE" ] && echo "DNS_Failed" > /dev/console || echo "BOOTED" > /dev/console

# Serve a file server that prints the hostname when requesting "/hostname"
mkdir /server
cd /server
hostname > hostname
# host it on python2 and fallback to python3
python -m SimpleHTTPServer 80
python3 -m http.server 80
