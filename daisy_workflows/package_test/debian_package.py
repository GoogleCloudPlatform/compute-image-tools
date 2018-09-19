#!/usr/bin/python
# Copyright 2018 Google Inc. All Rights Reserved.
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

"""Install specific compute-image-packages on debian

Parameters (retrieved from instance metadata):

git_revision: The git revision/branch from compute-image-packages to be tested
git_uri: The git uri reference to the compute-image-packages repository
"""

import glob
import logging
import os

import utils

# The following package is necessary to retrieve from https repositories.
utils.AptGetInstall([
    'git', 'python-setuptools', 'devscripts', 'dh-systemd', 'python-all',
    'python3-all', 'python3-setuptools', 'python-pytest', 'python3-pytest',
    'python-mock'])


def main():
  # Get Parameters.
  git_revision = utils.GetMetadataAttribute(
      'git_revision', raise_on_not_found=True)
  git_uri = utils.GetMetadataAttribute('git_uri', raise_on_not_found=True)
  pkg_folder = 'compute-image-packages'

  logging.info('git uri: %s' % git_uri)
  logging.info('git revision: %s' % git_revision)

  # get sources
  utils.Execute(['git', 'clone', '-n', git_uri, pkg_folder])
  os.chdir(pkg_folder)
  utils.Execute(['git', 'checkout', git_revision])

  # get version for generating the package
  rc, output = utils.Execute(
      ['python', 'setup.py', '--version'], capture_output=True)
  version = output.strip()
  os.chdir('..')
  utils.Execute(
      ['tar', '--exclude', '.git', '-czvf',
       'google-compute-image-packages_%s.orig.tar.gz' % version,
       pkg_folder])
  os.chdir(pkg_folder)

  # generate debian packages
  utils.Execute(['debuild', '-us', '-uc'])
  packages = map(os.path.abspath, glob.glob('../*.deb'))

  # now oslogin
  os.chdir('google_compute_engine_oslogin')
  utils.Execute(['./packaging/setup_deb.sh'])
  packages += glob.glob('/tmp/debpackage/*.deb')

  # install all generated packages
  utils.Execute(
      ['apt-get', 'install', '-y', '--reinstall', '--allow-downgrades'] +
      packages)


if __name__ == '__main__':
  try:
    main()
    logging.success('Debian package installation was successful!')
  except Exception as e:
    logging.error('Debian package installation failed: %s' % e)
