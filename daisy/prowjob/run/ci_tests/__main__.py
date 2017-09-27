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
import multiprocessing
import os
import re
import requests
import sys
import tarfile
import time

import google.auth
from google.auth.transport.requests import AuthorizedSession

from run import common
from run import git
from run import gcs
from run import logging


ARGS = []
PARALLEL_TESTS = 10
REPO_OWNER = 'GoogleCloudPlatform'
REPO_NAME = 'compute-image-tools'
REPO_URL = 'https://github.com/%s/%s.git' % (REPO_OWNER, REPO_NAME)
TEST_PROJECT = 'gce-daisy-test'
TEST_ID = str(common.utc_timestamp())
TEST_BUCKET = gcs.BUCKET
TEST_GCS_DIR = common.urljoin('tests', TEST_ID)
WFS_TAR = 'wfs.tar.gz'
TEST_WFS_GCS = common.urljoin(TEST_GCS_DIR, WFS_TAR)

BUILD_API_URL = 'https://cloudbuild.googleapis.com/v1'
session = None
suite_rgx = re.compile(r'(?P<suite>.*[^\d])(?P<test_num>\d*)\.wf\.json$')


def build_wf_suites(wfs):
    suites = {}
    for wf in wfs:
        match = suite_rgx.match(wf)
        if not match:
            continue
        suite = match.group('suite')
        test_num = match.group('test_num')
        test_num = int(test_num) if test_num else 0
        suites[suite] = suites.get(suite, []) + [(test_num, wf)]

    for suite in suites:
        suites[suite] = sorted(suites[suite])

    return suites


def main():
    logging.info('Got --tests=%s', ARGS.tests)

    logging.info('Fetching Daisy Repo.')
    repo = git.Repo('repo', clone=REPO_URL)
    if repo.clone_code:
        return repo.clone_code

    logging.info('Tar\'ing workflows to upload to GCS.')
    wf_dir = os.path.join('repo', 'daisy_workflows')
    with tarfile.open('wfs.tar.gz', 'w:gz') as tgz:
        tgz.add(wf_dir, arcname=os.path.sep)

    logging.info('Uploading tests to %s.', TEST_WFS_GCS)
    gcs.upload_file('wfs.tar.gz', TEST_WFS_GCS, 'application/gzip')

    logging.info('Running test workflows.')
    wfs = glob.glob(os.path.join(wf_dir, ARGS.tests, '*.wf.json'))
    wfs = [os.path.join(ARGS.tests, os.path.basename(wf)) for wf in wfs]
    suites = build_wf_suites(wfs)
    pool = multiprocessing.Pool(PARALLEL_TESTS)
    r = pool.map(run_suite, suites.values())
    return int(any(r))


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


def run_suite(suite):
    for _, wf in suite:
        logging.info('Running test %s', wf)

        body = {
            'source': {
                'storageSource': {
                    'bucket': TEST_BUCKET,
                    'object': TEST_WFS_GCS,
                }
            },
            'logsBucket': common.urljoin(TEST_BUCKET, TEST_GCS_DIR),
            'steps': [{
                'name': 'gcr.io/compute-image-tools/daisy:latest',
                'args': [
                    '-var:test-id=%s' % TEST_ID,
                    wf,
                ],
            }]
        }
        method = common.urljoin('projects', TEST_PROJECT, 'builds')
        resp = session.post(common.urljoin(BUILD_API_URL, method), json=body)
        try:
            resp.raise_for_status()
        except requests.exceptions.HTTPError as e:
            logging.error('Error creating test build %s: %s', wf, e)
            return 1

        op_name = resp.json()['name']
        data = {}
        while not data.get('done', False):
            time.sleep(5)
            resp = session.get(common.urljoin(BUILD_API_URL, op_name))
            try:
                resp.raise_for_status()
            except requests.exceptions.HTTPError as e:
                logging.error('Error getting test %s status: %s', wf, e)
                return 1
            data = resp.json()

        if 'error' in data:
            logging.error('Test %s failed: %s', wf, data['error'])
            return 1
        else:
            logging.info('Test %s finished successfully.', wf)
    return 0


def setup_session():
    scopes = ['https://www.googleapis.com/auth/cloud-platform']
    creds, _ = google.auth.default(scopes)
    return AuthorizedSession(creds)


if __name__ == '__main__':
    ARGS = parse_args()
    session = setup_session()

    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
