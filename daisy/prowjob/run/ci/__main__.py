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
import multiprocessing
import os
import re
import requests
import sys
import tarfile
import time
import xml.etree.ElementTree

import google.auth
from google.auth.transport.requests import AuthorizedSession
from junit_xml import TestCase, TestSuite

from run import common
from run import constants
from run import git
from run import logging
from run import result


ARGS = []
OTHER_ARGS = []
PARALLEL_TESTS = 10
REPO_URL = 'https://github.com/%s/%s.git' % (
        constants.REPO_OWNER, constants.REPO_NAME)
TEST_ID = str(common.utc_timestamp())
WFS_TAR = 'wfs.tar.gz'

BUILD_API_URL = 'https://cloudbuild.googleapis.com/v1'
session = None
suite_rgx = re.compile(r'(?P<suite>.*[^\d])(?P<test_num>\d*)\.wf\.json$')

TGZ_NAME = 'wfs.tar.gz'

build_log = StringIO()
build_log_handler = logging.StreamHandler(build_log)
build_log_handler.setLevel(logging.INFO)
logging.getLogger().addHandler(build_log_handler)

res = None


def build_subsuites(wfs):
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

    setup_result(repo.commit)

    logging.info('Tar\'ing workflows to upload to GCS.')
    wf_dir = os.path.join(repo.root, 'daisy_workflows')
    with tarfile.open(TGZ_NAME, 'w:gz') as tgz:
        tgz.add(wf_dir, arcname=os.path.sep)
    res.artifact(TGZ_NAME, path=TGZ_NAME, content_type='application/gzip')

    logging.info('Running test workflows.')
    wfs = glob.glob(os.path.join(wf_dir, ARGS.tests, '*.wf.json'))
    wfs = [os.path.join(ARGS.tests, os.path.basename(wf)) for wf in wfs]
    subsuites = build_subsuites(wfs)
    pool = multiprocessing.Pool(PARALLEL_TESTS)
    res.started()
    test_results = pool.map(run_subsuite, subsuites.values())
    code = 0
    test_cases = []
    for r in test_results:
        code = code or r[0]
        test_cases.extend(r[1])
    res.finished('FAILURE' if code else 'SUCCESS')
    ts = TestSuite(ARGS.tests, test_cases)
    ts_xml = xml_add_testcase_ids(ts.build_xml_doc(), ts)
    ts_data = xml.etree.ElementTree.tostring(ts_xml)
    res.artifact('junit_0.xml', data=ts_data, content_type='application/xml')
    res.build_log(build_log.getvalue())
    return code


def parse_args(arguments=None):
    """Parse arguments or sys.argv[1:]."""
    p = argparse.ArgumentParser()
    p.add_argument(
            '--tests', required=True,
            help=('The test workflows to run. The workflows are run at repo '
                  'HEAD.'))
    p.add_argument(
            '--version', default='latest', choices=['latest', 'release'],
            help='The image version to run tests against.')

    args, other_args = p.parse_known_args(arguments)
    return args, other_args


def run_subsuite(suite):
    code = 0
    test_cases = []
    for _, wf in suite:
        start = time.time()
        wf_return_code, testcase_id = run_wf(wf)
        end = time.time()
        tc = TestCase(wf, ARGS.tests, end - start)
        if testcase_id:
            tc.id = testcase_id
            tc.log_url = common.urljoin(
                    constants.GCS_API_BASE, constants.BUCKET, res.base_path,
                    'artifacts', 'log-%s.txt' % testcase_id)
        if wf_return_code:
            tc.add_failure_info('Failed with code %s' % wf_return_code)
        test_cases.append(tc)
        code = code or wf_return_code
    return code, test_cases


def run_wf(wf):
    args = OTHER_ARGS + ['-var:test-id=%s' % TEST_ID, wf]
    logging.info('Running test %s with args %s', wf, args)

    artifacts_path = common.urljoin(res.base_path, 'artifacts')
    body = {
        'source': {
            'storageSource': {
                'bucket': constants.BUCKET,
                'object': common.urljoin(artifacts_path, TGZ_NAME),
            }
        },
        'logsBucket': common.urljoin(constants.BUCKET, artifacts_path),
        'steps': [{
            'name': 'gcr.io/compute-image-tools/daisy:%s' % ARGS.version,
            'args': args,
        }],
        'timeout': '36000s',
    }
    method = common.urljoin('projects', constants.TEST_PROJECT, 'builds')
    resp = session.post(common.urljoin(BUILD_API_URL, method), json=body)
    try:
        resp.raise_for_status()
    except requests.exceptions.HTTPError as e:
        logging.error('Error creating test build %s: %s', wf, e)
        return 1, None

    op_data = resp.json()
    testcase_id = op_data['metadata']['build']['id']
    data = {}
    while not data.get('done', False):
        time.sleep(5)
        resp = session.get(common.urljoin(BUILD_API_URL, op_data['name']))
        try:
            resp.raise_for_status()
        except requests.exceptions.HTTPError as e:
            logging.error('Error getting test %s status: %s', wf, e)
            return 1, testcase_id
        data = resp.json()

    if 'error' in data:
        logging.error('Test %s failed: %s', wf, data['error'])
        return 1, testcase_id
    else:
        logging.info('Test %s finished successfully.', wf)
        return 0, testcase_id


def setup_result(commit):
    global res
    res = result.Periodic(commit)


def setup_session():
    scopes = ['https://www.googleapis.com/auth/cloud-platform']
    creds, _ = google.auth.default(scopes)
    return AuthorizedSession(creds)


def xml_add_testcase_ids(ts_xml, ts):
    for tc_xml, tc in zip(ts_xml, ts.test_cases):
        tc_xml.attrib['id'] = tc.id
    return ts_xml


if __name__ == '__main__':
    ARGS, OTHER_ARGS = parse_args()
    session = setup_session()

    return_code = main()
    logging.info('main() returned with code %s', return_code)
    logging.shutdown()
    sys.exit(return_code)
