# Copyright 2019 Google Inc. All Rights Reserved.
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
FROM golang

# Build test runner
WORKDIR /cli_tools_e2e_test
COPY cli_tools_e2e_test/ .
RUN cd gce_image_import_export && CGO_ENABLED=0 go build -o /gce_image_import_export_test_runner
RUN chmod +x /gce_image_import_export_test_runner

# Build binaries to test
WORKDIR /cli_tools
COPY cli_tools/ .
RUN cd gce_vm_image_import && CGO_ENABLED=0 go build -o /gce_vm_image_import
RUN chmod +x /gce_vm_image_import
RUN cd gce_vm_image_export && CGO_ENABLED=0 go build -o /gce_vm_image_export
RUN chmod +x /gce_vm_image_export

# Build test container
FROM gcr.io/compute-image-tools-test/wrapper-with-gcloud:latest
ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /gce_image_import_export_test_runner gce_image_import_export_test_runner
COPY --from=0 /gce_vm_image_import gce_vm_image_import
COPY --from=0 /gce_vm_image_export gce_vm_image_export
COPY /daisy_workflows/ /daisy_workflows/
ENTRYPOINT ["./wrapper", "./gce_image_import_export_test_runner"]
