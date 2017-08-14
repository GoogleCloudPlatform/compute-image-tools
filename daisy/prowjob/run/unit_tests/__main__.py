#!/usr/bin/env python
# Copyright 2017 Google Compute Engine Guest OS team.
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
import subprocess
import sys

from run import logging

ARGS = None
GCS_BKT = 'gce-daisy-test'
GIT_OWNER = 'GoogleCloudPlatform'
GIT_REPO = 'compute-image-tools'
CLONE_DIR = 'repo'


def call(cmd, cwd=os.getcwd()):
    logging.info('Running %q in directory %q.', ' '.join(cmd), cwd)
    return subprocess.call(cmd, cwd=cwd)


def main():
    logging.info('Fetching Daisy.')
    url = os.path.join('github.com/', GIT_OWNER, GIT_REPO)
    code = call(['go', 'get', url])
    if code:
        return code

    logging.info('Checking out PR #%s', ARGS.pr)
    cwd = os.path.join(os.getenv('GOPATH'), 'src', url)
    code = call(['git', 'checkout', '-b', 'test'], cwd)
    if code:
        return code
    code = call(['git', 'pull', 'origin', 'pull/%s/head' % ARGS.pr], cwd)
    if code:
        return code

    logging.info('Running tests.')
    cwd = os.path.join(cwd, 'daisy')
    code = call(['go', 'test', './...'], cwd)
    return code


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument('--pr', required=True, help="The pull request id.")
    args, _ = p.parse_known_args(arguments)
    return args

if __name__ == '__main__':
    ARGS = parse_args()
    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
