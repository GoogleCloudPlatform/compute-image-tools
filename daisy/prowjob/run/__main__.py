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
from run import common
from run import gcs
from run import logging

# Nones will be set before main() is called.
ARGS = None
OTHER_ARGS = None
RUN_ID = None
RUN_LOG = 'run.log'
ACTION_OUT = 'action.stdout'
ACTION_ERR = 'action.stderr'
OBJ_DIR = None

logfile_handler = logging.FileHandler(filename=RUN_LOG)
logfile_handler.setLevel(level=logging.INFO)
logging.getLogger().addHandler(logfile_handler)

epoch = common.utc_timestamp()


def main():
    cmd = ['python', '-m', ARGS.action] + OTHER_ARGS
    cwd = os.path.dirname(os.path.realpath(__file__))

    with open(ACTION_OUT, 'w') as out:
        with open(ACTION_ERR, 'w') as err:
            p = call(cmd, cwd=cwd, stdout=out, stderr=err)
    logging.info('Return code = %s', p.returncode)

    logging.info('Uploading logs to gs://%s/%s/', gcs.BUCKET, OBJ_DIR)
    gcs_action_out = os.path.join(OBJ_DIR, ACTION_OUT)
    gcs_action_err = os.path.join(OBJ_DIR, ACTION_ERR)
    gcs.upload_file(ACTION_OUT, gcs_action_out, 'text/plain')
    gcs.upload_file(ACTION_ERR, gcs_action_err, 'text/plain')
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
        gcs.upload_file(RUN_LOG, os.path.join(OBJ_DIR, 'run.log'), 'text/plain')
