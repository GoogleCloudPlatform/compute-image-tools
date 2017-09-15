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
from Crypto import Random
import glob
import multiprocessing
import os
import sys
import tarfile
import time

import apiclient
from oauth2client.service_account import ServiceAccountCredentials

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
TEST_GCS_DIR = os.path.join('tests', TEST_ID)
WFS_TAR = 'wfs.tar.gz'
TEST_WFS_GCS = os.path.join(TEST_GCS_DIR, WFS_TAR)

cloud_builder_client = None


def main():
    logging.info('Got --tests=%s', ARGS.tests)

    logging.info('Fetching Daisy Repo.')
    repo = git.Repo('repo', clone=REPO_URL)
    if repo.clone_code:
        return repo.clone_code

    logging.info('Tar\'ing workflows to upload to GCS.')
    wf_dir = os.path.join('repo', 'daisy_workflows', ARGS.tests)
    with tarfile.open('wfs.tar.gz', 'w:gz') as tgz:
        tgz.add(wf_dir, arcname=os.path.sep)

    logging.info('Uploading tests to %s.', TEST_WFS_GCS)
    gcs.upload_file('wfs.tar.gz', TEST_WFS_GCS, 'application/gzip')

    logging.info('Running test workflows.')
    wfs = glob.glob(os.path.join(wf_dir, '*.wf.json'))
    wfs = [os.path.basename(wf) for wf in wfs]
    pool = multiprocessing.Pool(PARALLEL_TESTS)
    r = pool.map(run_wf, wfs)
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


def run_wf(wf):
    logging.info('Running test %s', wf)

    # This is called by multiple processes and pycrypto requires that each
    # fork calls this.
    Random.atfork()

    body = {
        'source': {
            'storageSource': {
                'bucket': TEST_BUCKET,
                'object': TEST_WFS_GCS,
            }
        },
        'logsBucket': os.path.join(TEST_BUCKET, TEST_GCS_DIR),
        'steps': [{
            'name': 'gcr.io/compute-image-tools/daisy:latest',
            'args': [
                wf,
            ],
        }]
    }
    client = cloud_builder_client
    op = client.projects().builds().create(
            projectId=TEST_PROJECT, body=body).execute()
    op_name = op['name']
    while 'done' not in op or not op['done']:
        time.sleep(2)
        op = client.operations().get(name=op_name).execute() or op

    if 'error' in op:
        logging.error('Error running test %s: %s', wf, op['error'])
        return 1
    return 0


if __name__ == '__main__':
    ARGS = parse_args()
    creds_filepath = os.environ['GOOGLE_APPLICATION_CREDENTIALS']
    creds = ServiceAccountCredentials.from_json_keyfile_name(
            creds_filepath, ['https://www.googleapis.com/auth/cloud-platform'])
    cloud_builder_client = apiclient.discovery.build(
            'cloudbuild', 'v1', credentials=creds)
    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
