#!/bin/bash
# Copyright 2019 Google Inc. All Rights Reserved.
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

source packaging/common.sh 

rm -rf ${GOPATH}/src/github.com/GoogleCloudPlatform/compute-image-tools
sudo cp -r ../../../compute-image-tools/ /usr/share/gocode/src/github.com/GoogleCloudPlatform/

echo "Building package"
sudo su -c "GOPATH=${GOPATH} ${GO} get -d github.com/google/googet/goopack"
$GO run github.com/google/googet/goopack packaging/googet/google-osconfig-agent.goospec