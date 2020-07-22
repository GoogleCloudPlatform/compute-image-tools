# Copyright 2018 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM launcher.gcr.io/google/debian9
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -q -y qemu-utils gnupg ca-certificates
RUN echo "deb http://packages.cloud.google.com/apt gcsfuse-stretch main" > /etc/apt/sources.list.d/gcsfuse.list
# gcsfuse, installed using instructions from:
#  https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md
COPY gcsfuse-apt-key.gpg .
RUN apt-key add gcsfuse-apt-key.gpg
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -q -y gcsfuse
COPY linux/gce_vm_image_import /gce_vm_image_import
COPY daisy_workflows/ /daisy_workflows/

ENTRYPOINT ["/gce_vm_image_import"]
