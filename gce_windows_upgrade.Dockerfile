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

FROM gcr.io/distroless/base

COPY cli_tools/gce_windows_upgrade/upgrade_script_2008r2_to_2012r2.ps1 upgrade_script_2008r2_to_2012r2.ps1
COPY linux/gce_windows_upgrade /gce_windows_upgrade

ENTRYPOINT ["/gce_windows_upgrade"]
