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
import os

from run.call import call


class Repo(object):

    def __init__(self, root, clone=None):
        root = root if os.path.isabs(root) else os.path.join(os.getcwd(), root)
        self._root = root
        self.clone_code = None
        if clone:
            cmd = ['git', 'clone', clone, self._root]
            self.clone_code = call(cmd, cwd=self._root).returncode

    @property
    def root(self):
        return self._root

    def checkout(self, branch=None, commit=None, tag=None, pr=None):
        if branch or commit:
            cmd = ['git', 'checkout', branch or commit]
        elif tag:
            cmd = ['git', 'checkout', 'tags/%s' % tag]
        else:
            cmd = ['git', 'fetch', 'origin', 'pull/%s/head:%s' % (pr, pr)]
            p = call(cmd, cwd=self._root)
            if p.returncode:
                return p.returncode
            cmd = ['git', 'checkout', '%s' % pr]
        return call(cmd, cwd=self._root).returncode
