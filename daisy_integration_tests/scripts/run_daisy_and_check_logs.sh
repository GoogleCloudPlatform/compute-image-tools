#!/bin/bash
# Copyright 2018 Google Inc. All Rights Reserved.
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

URL='http://metadata/computeMetadata/v1/instance'
INSTANCE_ID="$(curl -f -H Metadata-Flavor:Google ${URL}/id)"
SHOULD_HAVE_LOGS="$(curl -f -H Metadata-Flavor:Google ${URL}/attributes/should_have_logs)"

install_daisy() {
  if ! gsutil cp gs://compute-image-tools/latest/linux/daisy .; then
    echo 'BuildFailed: Error pulling Daisy.'
    exit 1
  fi
  chmod +x ./daisy
}

run_daisy() {
  echo '
  {
    "Name": "create-disks-test",
    "Steps": {
     "create-disks": {
       "CreateDisks": [
         {
           "name": "disk-from-image-family-url",
           "sourceImage": "projects/debian-cloud/global/images/family/debian-10",
           "type": "pd-ssd"
         }
       ]
     }
    }
  }' >wf.json

  if ! ./daisy wf.json; then
    echo 'BuildFailed: Error executing Daisy.'
    exit 1
  fi
}

verify_logs() {
  echo 'BuildStatus: Waiting for logs to propagate to StackDriver.'
  sleep 120
  LOGS=$(gcloud logging read "resource.labels.instance_id=$INSTANCE_ID jsonPayload.workflow=create-disks-test")
  if [ "$SHOULD_HAVE_LOGS" = 'true' ]; then
    if [ -z "$LOGS" ]; then
      echo 'BuildFailed: Expected Cloud Logs.'
      return 1
    else
      echo 'Pass: Found expected Cloud Logs.'
    fi
  else
    if [ -n "$LOGS" ]; then
      echo 'BuildFailed: Expected no Cloud Logs.'
      return 1
    else
      echo 'Pass: Cloud Logs were not written.'
    fi
  fi
}

install_daisy
run_daisy
verify_logs

echo 'BuildSuccess: Daisy completed.'
