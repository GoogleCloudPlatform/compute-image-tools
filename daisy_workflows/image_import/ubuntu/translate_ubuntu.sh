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

# Download translate_el.py
URL="http://metadata/computeMetadata/v1/instance/attributes"
GCS_SOURCE_PATH="$(curl -f -H Metadata-Flavor:Google ${URL}/daisy-sources-path)"
gsutil cp "${GCS_SOURCE_PATH}/translate_ubuntu.py" /tmp/translate_ubuntu.py

# Install dependencies
DEBIAN_FRONTEND=noninteractive apt-get -y install python-guestfs python-pycurl tinyproxy

# cloud-init command uses GCE API, as libguestfs executes inside qemu, it uses a
# different network address and so GCE API is not accessible.
# Configure tinyproxy to provide httpproxy to the machine
# TODO: find a better solution, qemu should be able to forward this.
cat > /etc/tinyproxy/tinyproxy.conf << EOF
User tinyproxy
Group tinyproxy
Port 8888
Timeout 600
LogLevel Info
PidFile "/run/tinyproxy/tinyproxy.pid"
MaxClients 100
MinSpareServers 5
MaxSpareServers 20
StartServers 10
MaxRequestsPerChild 0
Allow 127.0.0.1
ViaProxyName "tinyproxy"
ConnectPort 443
ConnectPort 563
EOF
/etc/init.d/tinyproxy restart

# Customize image
python /tmp/translate_ubuntu.py
