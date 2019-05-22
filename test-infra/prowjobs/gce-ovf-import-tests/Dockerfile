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
FROM golang:alpine

RUN apk add --no-cache git

WORKDIR /
RUN go get -d github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests
RUN CGO_ENABLED=0 go build -o /gce_ovf_import_test_runner github.com/GoogleCloudPlatform/compute-image-tools/gce_ovf_import_tests
RUN chmod +x /gce_ovf_import_test_runner

FROM gcr.io/compute-image-tools-test/wrapper:latest

ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /gce_ovf_import_test_runner gce_ovf_import_test_runner
COPY /gce_ovf_import_tests/scripts/ /gce_ovf_import_tests/scripts/
COPY /daisy_integration_tests/scripts/ /daisy_integration_tests/scripts/
COPY /daisy_workflows/ /daisy_workflows/
ENTRYPOINT ["./wrapper", "./gce_ovf_import_test_runner"]
