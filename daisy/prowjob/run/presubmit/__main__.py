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

GOLINT_PACKAGE = 'github.com/golang/lint/golint'

PACKAGE = 'github.com/%s/%s/daisy' % (REPO_OWNER, REPO_NAME)
PACKAGE_PATH = os.path.join(os.environ['GOPATH'], 'src', PACKAGE)


def main():
    logging.info('Downloading Daisy repo.')
    code = call(['go', 'get', PACKAGE]).returncode
    if code:
        return code

    if ARGS.golint:
        logging.info('Downloading golint.')
        code = call(['go', 'get', GOLINT_PACKAGE]).returncode
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
    if ARGS.gofmt:
        code = call(['gofmt', '-l', '.'], cwd=PACKAGE_PATH).returncode
        if code:
            return code
    if ARGS.govet:
        code = call(['go', 'vet', './...'], cwd=PACKAGE_PATH).returncode
        if code:
            return code
    if ARGS.golint:
        cmd = ['golint', '-set_exit_status', './...']
        code = call(cmd, cwd=PACKAGE_PATH).returncode
        if code:
            return code
    if ARGS.gotest:
        cmd = ['go', 'test', '-race', './...']
        code = call(cmd, cwd=PACKAGE_PATH).returncode
        if code:
            return code
    return 0


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument('--pr', required=True, help='The pull request #.')
    g = p.add_argument_group('Checks')
    g.add_argument('--gofmt', action='store_true', help='Run `gofmt -l .`.')
    g.add_argument('--govet', action='store_true', help='Run `go vet ./...`.')
    g.add_argument('--golint', action='store_true', help='Run `golint ./...`.')
    g.add_argument(
            '--gotest', action='store_true',
            help='Run `go test -race ./...`.')
    args, _ = p.parse_known_args(arguments)
    if not any([args.gofmt, args.govet, args.golint, args.gotest]):
        msg = 'One or more of {--gofmt, --govet, --golint, --gotest} required.'
        p.error(msg)
    return args


if __name__ == '__main__':
    ARGS = parse_args()
    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
