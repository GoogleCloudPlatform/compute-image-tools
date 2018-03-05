#!/bin/sh
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

git clone https://github.com/${REPO_OWNER}/${REPO_NAME} /repo
cd /repo

# Pull PR if this is a PR.
if [ ! -z "${PULL_NUMBER}" ]; then
  git fetch origin pull/${PULL_NUMBER}/head:${PULL_NUMBER}
  git checkout ${PULL_NUMBER}
fi

flake8 --ignore E111,E114,E121,E125,E128,E129 --import-order-style=google /repo
RET=$?

# Print results and return.
if [ ${RET} != 0 ]; then
  echo "flake8 returned ${RET}"
  exit 1
fi
exit 0
