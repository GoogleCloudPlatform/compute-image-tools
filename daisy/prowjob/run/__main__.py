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
import time

import google.auth
import google.cloud.storage

from run import logging

# Nones will be set before main() is called.
ARGS = None
OTHER_ARGS = None
RUN_ID = None
RUN_LOG = 'run.log'
BUCKET = 'gce-daisy-test'
OBJ_DIR = None

logging.basicConfig(filename=RUN_LOG, level=logging.INFO)

epoch = int(time.time())
creds, proj = google.auth.default()
gcs = google.cloud.storage.Client(proj, creds)
bucket = gcs.bucket('gce-daisy-test')


def main():
    cmd = ['python', '-m', ARGS.action] + OTHER_ARGS
    cwd = os.path.dirname(os.path.realpath(__file__))

    logging.info('Running "%s" in directory "%s"', ' '.join(cmd), cwd)
    p = subprocess.Popen(
        cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = p.communicate()
    logging.info('Return code = %s', p.returncode)

    logging.info('Uploading logs to gs://%s/%s/', BUCKET, OBJ_DIR)
    obj = bucket.blob(os.path.join(OBJ_DIR, 'action_stdout'))
    obj.upload_from_string(out)
    obj = bucket.blob(os.path.join(OBJ_DIR, 'action_stderr'))
    obj.upload_from_string(err)
    return p.returncode


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument(
        '--action', required=True, choices=['e2e_tests', 'unit_tests'],
        help='The action to run: e2e_tests or unit_tests.')
    args, other_args = p.parse_known_args(arguments)
    return args, other_args

if __name__ == '__main__':
    try:
        ARGS, OTHER_ARGS = parse_args()
        RUN_ID = '_'.join([ARGS.action] + OTHER_ARGS + [str(epoch)])
        OBJ_DIR = os.path.join('prow_logs', RUN_ID)
        sys.exit(main())
    finally:
        logging.shutdown()
        obj = bucket.blob(os.path.join(OBJ_DIR, 'run.log'))
        obj.upload_from_filename(RUN_LOG, 'text/plain')
