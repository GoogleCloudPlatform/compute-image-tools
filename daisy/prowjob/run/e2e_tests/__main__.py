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
import glob
import os
import sys

from run.call import call
from run import git
from run import logging

ARGS = []
PACKAGE_URL = 'github.com/GoogleCloud/compute-image-tools'
PACKAGE_PATH = os.path.join(os.environ['GOPATH'], 'src', PACKAGE_URL)


def main():
    logging.info('Got --tests=%s', ARGS.tests)

    logging.info('Downloading Daisy')
    cmd = ['go', 'get', PACKAGE_URL]
    code = call(cmd).returncode
    if code:
        return code

    # commit = 'Something from Argo'
    repo = git.Repo(PACKAGE_PATH)
    repo.checkout(branch='master')

    wf_dir = os.path.join(PACKAGE_PATH, 'daisy_workflows', ARGS.tests)
    wfs = glob.glob(os.path.join(wf_dir, '*.wf.json'))

    logging.info('Running tests...')
    for wf in wfs:
        wf = os.path.join(ARGS.tests, os.path.basename(wf))
        logging.info('Running test %s', os.path.basename(wf))
    return 0


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument(
            '--tests', required=True,
            help=('The test workflows to run. The workflows are run at the '
                  'same commit from which the image was built.'))
    p.add_argument(
            '--version', default='latest', choices=['latest', 'release'],
            help='The image version to run e2e tests against.')

    args = p.parse_args(arguments)
    return args


if __name__ == '__main__':
    ARGS = parse_args()
    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
