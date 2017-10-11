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
import json
import os

import constants
import common
import gcs


class _Result(object):
    _path = None
    _version = None

    def artifact(self, data, filename):
        path = common.urljoin(self._path, 'artifacts', filename)
        gcs.upload_string(data, path)

    def build_log(self, data):
        path = common.urljoin(self._path, 'build-log.txt')
        gcs.upload_string(data, path, 'plain/text')

    def finished(self, result, metadata=None):
        path = common.urljoin(self._path, 'finished.json')
        data = json.dumps({
            'timestamp': common.utc_timestamp(),
            'result': result,
            'version': self._version,
            'metadata': metadata or {},
        })
        gcs.upload_string(data, path, 'application/json')

    def started(self):
        path = common.urljoin(self._path, 'started.json')
        data = {
            'timestamp': common.utc_timestamp(),
            'version': self._version,
        }
        if constants.PULL_REFS:
            data['pull'] = constants.PULL_REFS
        gcs.upload_string(json.dumps(data), path, 'application/json')


class Periodic(_Result):
    def __init__(self, version):
        build_num = constants.BUILD_NUM
        job_name = constants.JOB_NAME
        self._path = os.path.join('logs', job_name, build_num)
        self._version = version


class PR(_Result):
    def __init__(self, pr, version):
        build_num = constants.BUILD_NUM
        job_name = constants.JOB_NAME
        org_repo = '%s_%s' % (constants.REPO_OWNER, constants.REPO_NAME)
        self._path = os.path.join(
                'pr-logs', 'pull', org_repo, str(pr), job_name, build_num)
        self._version = version
