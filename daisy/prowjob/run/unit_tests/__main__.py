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

import argparse
import os
import sys

from run.call import call
from run import git
from run import logging

ARGS = None
REPO_OWNER = 'GoogleCloudPlatform'
REPO_NAME = 'compute-image-tools'

PACKAGE = 'github.com/%s/%s/daisy' % (REPO_OWNER, REPO_NAME)
PACKAGE_PATH = os.path.join(os.environ['GOPATH'], 'src', PACKAGE)


def main():
    logging.info('Fetching Daisy.')
    code = call(['go', 'get', PACKAGE]).returncode
    if code:
        return code

    logging.info('Checking out PR #%s', ARGS.pr)
    repo = git.Repo(PACKAGE_PATH)
    code = repo.checkout(pr=ARGS.pr)
    if code:
        return code

    logging.info('Pulling dependencies.')
    code = call(['go', 'get', '-t', './...'], cwd=PACKAGE_PATH).returncode
    if code:
        return code

    logging.info('Running checks.')
    code = call(['gofmt', '-l', '.'], cwd=PACKAGE_PATH).returncode
    if code:
        return code
    code = call(['go', 'vet', './...'], cwd=PACKAGE_PATH).returncode
    if code:
        return code
    return call(['go', 'test', '-race', './...'], cwd=PACKAGE_PATH).returncode


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument('--pr', required=True, help="The pull request #.")
    args, _ = p.parse_known_args(arguments)
    return args


if __name__ == '__main__':
    ARGS = parse_args()
    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
