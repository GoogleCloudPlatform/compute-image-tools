# Copyright 2020 Google Inc. All Rights Reserved.
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


FROM debian

# 1 - aws cli
RUN apt-get update
RUN apt-get -y install zip unzip curl
RUN curl https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -o awscliv2.zip
RUN unzip awscliv2.zip
RUN ./aws/install

# 2 - onestep-importer cli
COPY linux/gce_onestep_image_import /gce_onestep_image_import
COPY linux/gce_vm_image_import /gce_vm_image_import
COPY daisy_workflows/ /daisy_workflows/

ENTRYPOINT ["/gce_onestep_image_import"]
