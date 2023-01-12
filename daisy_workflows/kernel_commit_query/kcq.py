#!/usr/bin/env python3
# Copyright 2022 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Queries the presence of feature commits in a OS kernel.

Parameters retrieved from instance metadata:

filter_spec:              Filter patches and commits to a given path/spec,
                          it's converted to git's --include flag wherever
                          applicable.
upstream_kernel_version:  Upstream kernel version containing all the patches
                          in the catalog.
upstream_kernel_repo:     URL to the upstream kernel's git repository.
"""

import json
import logging
import os
import platform
import re
import subprocess as sp

import requests
import utils

upstream_kernel_dir = '/files/upstream_kernel'
distro_kernel_dir = '/files/distro_kernel'
patches_dir = '/files/patches'
result_file = '/files/result.json'
catalog_file = '/files/catalog'
changelog_file = '/files/kernel.changelog'
upstream_commit_url = 'https://git.kernel.org/pub/scm/linux/kernel/\
git/torvalds/linux.git/patch/?id='

devnull = {'stdout': sp.DEVNULL, 'stderr': sp.DEVNULL}
capture = {'stdout': sp.PIPE, 'stderr': sp.PIPE}


def run(args, stdout=None, stderr=None, cwd=None, check=True):
    return sp.run(args, cwd=cwd, check=check, stdout=stdout,
                  stderr=stderr)


class Attributes(object):
    def __init__(self):
        attrs = [
            'filter_spec',
            'upstream_kernel_version',
            'upstream_kernel_repo',
            'daisy-outs-path',
            'strategy',
        ]

        for curr in attrs:
            self.__initAttribute(curr)

        self.__initKernelVersion()
        self.__dump()

    def __initAttribute(self, attr, required=True):
        value = utils.GetMetadataAttribute(attr, raise_on_not_found=required)
        self.__dict__[attr.replace('-', '_')] = value

    def __initKernelVersion(self):
        uname = platform.uname()
        self.kernel_pkg_version = uname.release
        self.kernel_version = re.sub('-.*$', '', uname.release)

        # release .0 is cut off i.e: 5.15.0 -> 5.15 to be aligned with
        # upstream's versioning scheme
        self.kernel_version = re.sub('.0$', '', self.kernel_version)

    def __dump(self):
        for key in self.__dict__:
            logging.info('%s: %s' % (key.upper(), self.__dict__[key]))


class CommitQuery(object):
    def __init__(self, attrs):
        self.attrs = attrs

    def __fetchUpstreamKernel(self):
        logging.info('Fetching upstream kernel')

        url = self.attrs.upstream_kernel_repo
        dest = upstream_kernel_dir

        if not os.path.exists(dest):
            run(['git', 'clone', url, dest], **devnull)

    def __getPatchesData(self, cids):
        kernel_path = upstream_kernel_dir
        result = {}

        for cid in cids:
            output = run(['git', 'show', '--format="%h%n%H%n%s"', cid],
                       cwd=kernel_path, **capture)
            lines = output.stdout.decode('utf-8').splitlines()
            if not len(lines) or len(lines) < 4:
                raise Exception('Failed to query commit data')
            result[cid] = {'abbrev_hash': lines[0], 'hash': lines[1],
                    'subject': lines[2]}

        return result

    # exports the patch chain between the running system kernel's version
    # and the 'upstream_kernel_version' provided in the interface
    def __exportPatches(self, write_patch=True):
        kernel_path = upstream_kernel_dir
        version = self.attrs.kernel_version
        upstream_version = self.attrs.upstream_kernel_version
        filter_spec = self.attrs.filter_spec
        dest = patches_dir

        if filter_spec is None or filter_spec == '':
            filter_spec = '.'

        res = []
        ver_spec = 'v%s..v%s' % (version, upstream_version)
        log = run(['git', 'log', ver_spec, '--format=%H', filter_spec],
                  cwd=kernel_path, **capture)

        if not os.path.exists(dest):
            os.makedirs(dest, mode=0o755)

        for cid in log.stdout.decode('utf-8').splitlines():
            res.insert(0, cid)

            if not write_patch:
                continue

            patch_file = os.path.join(dest, cid)

            if os.path.exists(patch_file):
                logging.info('%s previously exported' % patch_file)
                continue

            logging.info('Exporting %s -> %s' % (cid, patch_file))
            patch = run(['git', 'format-patch', '--stdout', '-1', cid],
                        cwd=kernel_path, **capture)

            with open(patch_file, 'w') as f:
                f.write(patch.stdout.decode("utf-8"))

        return res

    # init and import distribution's kernel git repo
    def __prepareDistroKernel(self):
        logging.info('Preparing distro kernel dir')
        git_cmds = ['init', 'add *', 'commit -m "import"']
        for curr in git_cmds:
            run(['git'] + curr.split(' '), cwd=distro_kernel_dir,
                **devnull)

    def __parsePatch(self, cid, patch):
        res = {'hash': cid, 'abbrev_hash': cid[0:7]}
        for curr in patch.split('\n'):
            if curr.startswith('Subject:'):
                res['subject'] = curr.replace('Subject: ', '')
                break
        return res

    def ChangelogCheck(self):
        logging.info('Checking patches from: %s' % patches_dir)

        found = []
        not_found = []

        catalog = self.__readCatalog()

        for curr_hash in catalog:
            resp = requests.get(upstream_commit_url + curr_hash)

            if not resp.ok:
                raise Exception("Failed to query upstream commit")

            curr = self.__parsePatch(curr_hash, resp.text)
            test_attrs = ['hash', 'subject', 'abbrev_hash']

            for attr in test_attrs:
                res = run(['grep', curr[attr], changelog_file], check=False,
                          **devnull)
                if res.returncode == 0:
                    curr_found = True
                    break

            if curr_found:
                found.append(curr_hash)
            else:
                not_found.append(curr_hash)

        self.__writeResult({'found': found, 'not_found': not_found})

    def ApplyCheck(self):
        logging.info('Applying patches from: %s' % patches_dir)

        found = []
        not_found = []
        filter_args = []

        self.__prepareDistroKernel()
        self.__fetchUpstreamKernel()

        catalog = self.__readCatalog()
        chain = self.__exportPatches()

        for cid in chain:
            patch_path = os.path.join(patches_dir, cid)
            apply_cmd = ['git', 'apply', patch_path] + filter_args

            res = run(apply_cmd + ['--check'], check=False,
                      cwd=distro_kernel_dir, **devnull)

            if res.returncode == 0:
                run(apply_cmd, cwd=distro_kernel_dir, **devnull)
                status = run(['git', 'status'], cwd=distro_kernel_dir,
                             **capture)

                logging.info('Commit %s is not present' % cid)

                if 'nothing to commit' not in status.stdout.decode('utf-8'):
                    git_cmds = ['add *', 'commit -m %s' % cid]

                    for curr in git_cmds:
                        run(['git'] + curr.split(' '), cwd=distro_kernel_dir,
                            **devnull)

                if cid in catalog:
                    not_found.append(cid)
            else:
                logging.info('Commit %s is present' % cid)

                if cid in catalog:
                    found.append(cid)

        self.__writeResult({'found': found, 'not_found': not_found})

    def __writeResult(self, result):
        result['kernel_pkg_version'] = self.attrs.kernel_pkg_version
        result['kernel_version'] = self.attrs.kernel_version

        with open(result_file, 'w') as f:
            f.write(json.dumps(result, indent=4))

        logging.info('Result wrote to: %s' % result_file)

        dest = os.path.join(self.attrs.daisy_outs_path, 'result.json')
        run(['gcloud', 'alpha', 'storage', 'cp', result_file, dest])
        logging.info('Result uploaded to: %s' % dest)

    def __readCatalog(self):
        catalog = open(catalog_file, 'r')
        res = []

        while True:
            line = catalog.readline()
            if not line:
                break
            res.append(line.replace('\n', ''))

        catalog.close()
        return res


def main():
    logging.getLogger().setLevel(logging.INFO)

    cq = CommitQuery(Attributes())
    if cq.attrs.strategy == 'apply-check':
        cq.ApplyCheck()
    elif cq.attrs.strategy == 'changelog':
        cq.ChangelogCheck()
    else:
        raise Exception('Unknown strategy: %s' % cq.attrs.strategy)

    logging.info('KCQStatus: success')


if __name__ == '__main__':
  main()
