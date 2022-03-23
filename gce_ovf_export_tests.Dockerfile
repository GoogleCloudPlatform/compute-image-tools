# Copyright 2021 Google Inc. All Rights Reserved.
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
ARG PROJECT_ID=compute-image-tools-test
FROM golang:alpine

RUN apk add --no-cache git

# Build test runner
COPY / /build
WORKDIR /build
RUN cd /build/cli_tools_tests/e2e/gce_ovf_export && CGO_ENABLED=0 go build -o /gce_ovf_export_test_runner
RUN chmod +x /gce_ovf_export_test_runner

# Build binaries to test
RUN cd /build/cli_tools/gce_ovf_export && CGO_ENABLED=0 go build -o /gce_ovf_export
RUN chmod +x /gce_ovf_export

# Build test container
FROM gcr.io/$PROJECT_ID/e2e-test-base:latest
COPY --from=0 /gce_ovf_export_test_runner gce_ovf_export_test_runner
COPY --from=0 /gce_ovf_export gce_ovf_export
COPY /cli_tools_tests/e2e/gce_ovf_import/scripts/ /scripts/
COPY /daisy_integration_tests/scripts/ /daisy_integration_tests/scripts/
COPY /daisy_workflows/ /daisy_workflows/
ENTRYPOINT ["./gce_ovf_export_test_runner"]
