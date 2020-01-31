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

# Build binaries to test
FROM golang
WORKDIR /cli_tools
COPY cli_tools/ .
RUN cd gce_onestep_image_import && CGO_ENABLED=0 go build -o /gce_onestep_image_import
RUN chmod +x /gce_onestep_image_import

# Build container
FROM google/cloud-sdk:slim
# 1 - aws cli
RUN apt-get update
RUN apt-get -y install zip unzip curl
RUN curl "https://d1vvhvl2y92vvt.cloudfront.net/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
RUN unzip awscliv2.zip
RUN ./aws/install
# 2 - onestep-importer cli
COPY --from=0 /gce_onestep_image_import gce_onestep_image_import
COPY /daisy_workflows/ /daisy_workflows/

ENTRYPOINT ["/gce_onestep_image_import"]
