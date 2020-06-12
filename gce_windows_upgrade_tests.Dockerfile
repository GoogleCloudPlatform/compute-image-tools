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
FROM golang

# Build test runner
WORKDIR /cli_tools_e2e_test
COPY cli_tools_e2e_test/ .
RUN cd gce_windows_upgrade && CGO_ENABLED=0 go build -o /gce_windows_upgrade_test_runner
RUN chmod +x /gce_windows_upgrade_test_runner

# Build binaries to test
WORKDIR /cli_tools
COPY cli_tools/ .
RUN cd gce_windows_upgrade && CGO_ENABLED=0 go build -o /gce_windows_upgrade
RUN chmod +x /gce_windows_upgrade

# Build test container
FROM gcr.io/$PROJECT_ID/wrapper-with-gcloud:latest
ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /gce_windows_upgrade_test_runner gce_windows_upgrade_test_runner
COPY --from=0 /gce_windows_upgrade gce_windows_upgrade
COPY /cli_tools/gce_windows_upgrade/upgrade_script_2008r2_to_2012r2.ps1 upgrade_script_2008r2_to_2012r2.ps1
ENTRYPOINT ["./wrapper", "./gce_windows_upgrade_test_runner"]
