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
ARG PROJECT_ID=compute-image-tools-test
FROM golang

# Build test runner
COPY / /kaniko/build
RUN CGO_ENABLED=0 go build -C /kaniko/build/cli_tools_tests/e2e/gce_windows_upgrade -buildvcs=false -o /gce_windows_upgrade_test_runner
RUN chmod +x /gce_windows_upgrade_test_runner

# Build binaries to test
RUN CGO_ENABLED=0 go build -C /kaniko/build/cli_tools/gce_windows_upgrade -buildvcs=false -o /gce_windows_upgrade
RUN chmod +x /gce_windows_upgrade

# Build test container
FROM google/cloud-sdk:debian_component_based
COPY --from=0 /gce_windows_upgrade_test_runner gce_windows_upgrade_test_runner
COPY --from=0 /gce_windows_upgrade gce_windows_upgrade
COPY /cli_tools/gce_windows_upgrade/upgrade_script.ps1 upgrade_script.ps1
ENTRYPOINT ["./gce_windows_upgrade_test_runner"]
