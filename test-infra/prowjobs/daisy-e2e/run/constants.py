# Copyright 2017 Google Inc. All Rights Reserved.
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

import os as _os

import run.common

BUCKET = 'compute-image-tools-test'
BUILD_API_URL = 'https://cloudbuild.googleapis.com/v1'
BUILD_NUM = _os.environ['BUILD_NUMBER']
GCS_API_BASE = 'https://storage.googleapis.com'
JOB_NAME = _os.environ['JOB_NAME']
PARALLEL_TESTS = 10
PULL_REFS = _os.getenv('PULL_REFS')
REPO_OWNER = _os.environ['REPO_OWNER']
REPO_NAME = _os.environ['REPO_NAME']
REPO_URL = 'https://github.com/%s/%s.git' % (REPO_OWNER, REPO_NAME)
TEST_ID = str(run.common.unix_time())
TEST_PROJECT = 'compute-image-tools-test'
