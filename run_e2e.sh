#!/bin/bash
# Copyright 2021 Google Inc. All Rights Reserved.
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

# Runs an e2e test suites using docker from the local machine. To see the
# available suites, run `ls *tests.Dockerfile`.
#
# Requirements:
#  1. Install gcloud and docker
#  2. Set your development test project using `gcloud config set project <value>`
#  3. Set the zone to execute tests using `gcloud config set compute/zone <value>`
#
# Usage:
#   ./run_e2e.sh <dockerfile>
#
# Options:
#  -test_suite_filter: optional regex pattern
#  -test_case_filter: optional regex pattern
#
#     When a filter is provided, only suites and cases that match the filter
#     will be executed. When filters are omitted, all tests are run.
#
#     Search for invocations of `junitxml.NewTestSuite` and `junitxml.NewTestCase` in
#     cli_tools_tests/e2e to find values.
#
# Examples:
#  Run ubuntu import tests:
#    ./run_e2e.sh gce_vm_image_import.Dockerfile \
#         -test_suite_filter=^ImageImport$ -test_case_filter=ubuntu
#
# Notes:
#  Tests may require resources in your project, such as networks, subnets, GCS objects,
#  GCS buckets, and PD images. If a resource is missing, you will need to manually
#  create it in order to run the test.

function print_header() {
  cat <<EOF
#
#  $1
#

EOF
}

cd "$(dirname "$0")"

#
# Create log directory.
#
print_header "Initializing logs directory"
LOGS_DIR=$(mktemp -d)
printf "Created %s\n\n" "$LOGS_DIR"

#
# Validate ZONE, PROJECT, and DOCKER_FILE.
#
print_header "Checking Environment"
ZONE=$(gcloud config get-value compute/zone | tr -d '\n')
if [ -z "$ZONE" ]; then
  echo "Set zone with 'gcloud config set compute/zone <value>'"
  exit 1
fi

PROJECT=$(gcloud config get-value project | tr -d '\n')
if [ -z "$PROJECT" ]; then
  echo "Set project with 'gcloud config set project <value>'"
  exit 1
fi

DOCKER_FILE=$1 && shift
if [ -z "$DOCKER_FILE" ]; then
  echo "Specify the dockerfile to run as the first argument"
  exit 1
fi
if [[ ! -f "$DOCKER_FILE" ]]; then
  echo "$DOCKER_FILE not found."
  exit 1
fi

#
# Check for gcloud and docker
#
echo "Checking for docker"
if ! docker --version >"$LOGS_DIR/docker-version.log" 2>&1; then
  echo "docker not found on the system path. Please install then re-run this script."
  exit 1
fi
echo " .. done"

echo "Checking for gcloud"
if ! gcloud --version >"$LOGS_DIR/gcloud-version.log" 2>&1; then
  echo "gcloud not found on the system path. Please install then re-run this script."
  exit 1
fi
echo " .. done"

#
# Run test suite using Docker.
#
cat <<EOF
Starting tests. Environment:
  project=$PROJECT
  zone=$ZONE
  logs=$LOGS_DIR
If there are authentication errors, run:
  gcloud auth login \\
   && gcloud auth configure-docker \\
   && gcloud auth application-default login

EOF
set -e
print_header "Building tests"
docker build -f "$DOCKER_FILE" . -t e2e 2>&1 | tee "$LOGS_DIR/build.log"

print_header "Running tests"
docker run --env GOOGLE_APPLICATION_CREDENTIALS= \
  --env CLOUDSDK_CONFIG=/root/.config/gcloud \
  -v "$HOME/.config/gcloud:/root/.config/gcloud" \
  e2e \
  -test_project_id="$PROJECT" \
  -test_zone="$ZONE" "$@" 2>&1 | tee "$LOGS_DIR/run.log"
print_header "Done"
echo "See $LOGS_DIR for logs."
