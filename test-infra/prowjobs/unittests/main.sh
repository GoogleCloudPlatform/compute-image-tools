#!/bin/bash
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

set -e
set -x

export GOCOVPATH=/gocov.txt
export PYCOVPATH=/pycov.txt

# Check this out in GOPATH since go package handling requires it to be here.
git clone https://github.com/$REPO_OWNER/$REPO_NAME $GOPATH/src/github.com/$REPO_OWNER/$REPO_NAME
cd $GOPATH/src/github.com/$REPO_OWNER/$REPO_NAME

# Pull PR if this is a PR.
if [ ! -z "$PULL_NUMBER" ]; then
  git fetch origin pull/$PULL_NUMBER/head:$PULL_NUMBER
  git checkout $PULL_NUMBER
fi

set +e

# Execute all unittests.sh scripts in their directories.
# Each script should APPEND coverage data to GOCOVPATH or PYCOVPATH.
RET=0
find . -type f -name "unittests.sh" | while read script; do
  # Change to the containing directory and run script.
  cd $(dirname $script)
  ./$(basename $script)

  UNITTEST_RET=$?
  if [ $RET == 0 ]; then
    RET=$UNITTEST_RET
  fi

  popd
done

set +x

# Upload coverage results to Codecov.
CODEV_COV_ARGS="-v -t $(cat $CODECOV_TOKEN) -B master -C $(git rev-parse HEAD)"
if [ ! -z "$PULL_NUMBER" ]; then
  CODEV_COV_ARGS="$CODEV_COV_ARGS -P $PULL_NUMBER"
fi
if [ -e $GOCOVPATH ]; then
  bash <(curl -s https://codecov.io/bash) -f $GOCOVPATH -F go_unittests $CODEV_COV_ARGS
fi
if [ -e $PYCOVPATH ]; then
  bash <(curl -s https://codecov.io/bash) -f $PYCOVPATH -F py_unittests $CODEV_COV_ARGS
fi
exit $RET
