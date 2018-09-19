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

"""Install specific compute-image-packages on centos/rhel

Parameters (retrieved from instance metadata):

git_revision: The git revision/branch from compute-image-packages to be tested
git_uri: The git uri reference to the compute-image-packages repository
"""

import glob
import logging
import os

import utils

# The following packages are necessary to retrieve from https repositories.
utils.YumInstall(['git', 'rpmdevtools', 'python-setuptools', 'python2-devel'])


def main():
  # Get Parameters.
  git_revision = utils.GetMetadataAttribute(
      'git_revision', raise_on_not_found=True)
  git_uri = utils.GetMetadataAttribute('git_uri', raise_on_not_found=True)

  logging.info('git uri: %s' % git_uri)
  logging.info('git revision: %s' % git_revision)

  # get sources
  pkg_folder = 'compute-image-packages'
  utils.Execute(['git', 'clone', '-n', git_uri, pkg_folder])
  os.chdir(pkg_folder)
  utils.Execute(['git', 'checkout', git_revision])

  # get version for generating the package
  rc, output = utils.Execute(
      ['python', 'setup.py', '--version'], capture_output=True)
  version = output.strip()
  os.chdir('..')

  # prepare source directory for rpm build
  src_folder = 'SOURCES'
  os.mkdir(src_folder)
  src_folder = os.path.abspath(src_folder)
  utils.Execute(
      ['tar', '--exclude', '.git', '-czvf',
       '%s/google-compute-engine_%s.orig.tar.gz' % (
           src_folder, version),
       pkg_folder])
  pkg_folder = os.path.abspath(pkg_folder)

  # generate rpm packages
  utils.Execute(
      ['rpmbuild', '--define', '_topdir %s' % os.getcwd(), '-ba'] +
      glob.glob('%s/specs/*' % pkg_folder))
  packages = map(os.path.abspath, glob.glob('RPMS/noarch/*.rpm'))

  # now oslogin
  os.chdir(os.path.join(pkg_folder, 'google_compute_engine_oslogin'))
  utils.Execute(['./packaging/setup_rpm.sh'])
  packages += glob.glob('/tmp/rpmpackage/*/rpmbuild/RPMS/x86_64/*.rpm')

  # install all generated packages
  utils.Execute(
      ['rpm', '-i', '--replacepkgs', '--oldpackage', '--replacefiles'] +
      packages)


if __name__ == '__main__':
  try:
    main()
    logging.success('RPM package installation was successful!')
  except Exception as e:
    logging.error('RPM package installation failed: %s' % e)
