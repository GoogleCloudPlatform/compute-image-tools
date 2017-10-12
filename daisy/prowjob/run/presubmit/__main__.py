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
from cStringIO import StringIO
import glob
import os
import sys

from run import constants
from run import git
from run import logging
from run import result
from run.call import call

ARGS = None

GOLINT_PACKAGE = 'github.com/golang/lint/golint'
GOJUNIT_PACKAGE = 'github.com/jstemmer/go-junit-report'
THIS_DIR = os.path.dirname(os.path.realpath(__file__))

build_log = StringIO()
build_log_handler = logging.StreamHandler(build_log)
build_log_handler.setLevel(logging.INFO)
logging.getLogger().addHandler(build_log_handler)


def main():
    logging.info(os.getcwd())
    logging.info('Downloading Daisy repo.')
    code = call(['go', 'get', constants.GOPACKAGE]).returncode
    if code:
        return code

    if ARGS.golint:
        logging.info('Downloading golint.')
        code = call(['go', 'get', GOLINT_PACKAGE]).returncode
        if code:
            return code

    if ARGS.gotest:
        logging.info('Downloading go-junit-report.')
        code = call(['go', 'get', GOJUNIT_PACKAGE]).returncode
        if code:
            return code

    logging.info('Checking out PR #%s', ARGS.pr)
    repo = git.Repo(constants.GOPACKAGE_PATH)
    code = repo.checkout(pr=ARGS.pr)
    if code:
        return code

    logging.info('Pulling dependencies.')
    code = call(['go', 'get', '-t', './...'], cwd=repo.root).returncode
    if code:
        return code

    logging.info('Running checks.')
    if ARGS.gofmt:
        code = call(['gofmt', '-l', '.'], cwd=repo.root).returncode
        if code:
            return code
    if ARGS.govet:
        code = call(['go', 'vet', './...'], cwd=repo.root).returncode
        if code:
            return code
    if ARGS.golint:
        cmd = ['golint', '-set_exit_status', './...']
        code = call(cmd, cwd=repo.root).returncode
        if code:
            return code
    if ARGS.gotest:
        commit = repo.commit
        package = constants.GOPACKAGE
        res = result.PR(ARGS.pr, commit)
        res.started()
        cmd = ['./unit_tests.sh', package, commit, str(ARGS.pr)]
        code = call(cmd, cwd=THIS_DIR).returncode
        if code:
            res.finished('FAILURE')
            res.build_log(build_log.read())
            return code
        res.finished('SUCCESS')

        for path in glob.glob(os.path.join(THIS_DIR, '*.xml')):
            name = os.path.basename(path)
            with open(path) as f:
                data = f.read()
            res.artifact('junit_%s' % name, data=data, content_type='application/xml')

        res.build_log(build_log.getvalue())

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
