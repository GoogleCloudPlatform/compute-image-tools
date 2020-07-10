#!/usr/bin/env python3
# Copyright 2017 Google Inc. All Rights Reserved.
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

"""Build the Debian image on a GCE VM.

Parameters (retrieved from instance metadata):

bootstrap_vz_manifest: The version of bootstrap-vz to retrieve and use.
bootstrap_vz_version: The version of bootstrap-vz to retrieve and use.
google_cloud_repo: The repo to use to build Debian. Must be one of
  ['stable' (default), 'unstable', 'staging'].
image_dest: The Cloud Storage destination for the resultant image.
"""
import collections
from datetime import datetime, timezone
import json
import logging
import os
import shutil
import urllib.request
import zipfile

import utils

utils.AptGetInstall(
    ['git', 'python-pip', 'qemu-utils', 'parted', 'kpartx', 'debootstrap',
     'python-yaml', 'python3-yaml'])
utils.PipInstall(
    ['termcolor', 'fysom', 'jsonschema', 'docopt', 'json_minify'])

import yaml  # noqa: E402,I202

BVZ_DIR = '/bvz'
REPOS = ['stable', 'unstable', 'staging']


def main():
  # Get Parameters.
  bvz_manifest = utils.GetMetadataAttribute(
      'bootstrap_vz_manifest', raise_on_not_found=True)
  bvz_url = utils.GetMetadataAttribute(
      'bootstrap_vz_version', raise_on_not_found=True)
  repo = utils.GetMetadataAttribute('google_cloud_repo',
                                    raise_on_not_found=True).strip()
  image_meta = utils.GetMetadataAttribute('image_meta',
                                          raise_on_not_found=True)
  image_dest = utils.GetMetadataAttribute('image_dest',
                                          raise_on_not_found=True)
  outs_path = utils.GetMetadataAttribute('daisy-outs-path',
                                         raise_on_not_found=True)

  if repo not in REPOS:
    raise ValueError(
        'Metadata "google_cloud_repo" must be one of %s.' % REPOS)

  logging.info('Bootstrap_vz manifest: %s' % bvz_manifest)
  logging.info('Bootstrap_vz URL: %s' % bvz_url)
  logging.info('Google Cloud repo: %s' % repo)

  # Download and setup bootstrap_vz.
  bvz_zip_dir = 'bvz_zip'
  urllib.request.urlretrieve(bvz_url, 'bvz.zip')
  with zipfile.ZipFile('bvz.zip', 'r') as z:
    z.extractall(bvz_zip_dir)
  logging.info('Downloaded and extracted %s to bvz.zip.' % bvz_url)
  bvz_zip_contents = [d for d in os.listdir(bvz_zip_dir)]
  bvz_zip_subdir = os.path.join(bvz_zip_dir, bvz_zip_contents[0])
  utils.Execute(['mv', bvz_zip_subdir, BVZ_DIR])
  logging.info('Moved bootstrap_vz from %s to %s.' % (bvz_zip_subdir, BVZ_DIR))
  bvz_bin = os.path.join(BVZ_DIR, 'bootstrap-vz')
  utils.MakeExecutable(bvz_bin)
  logging.info('Made %s executable.' % bvz_bin)
  bvz_manifest_file = os.path.join(BVZ_DIR, 'manifests', bvz_manifest)

  # Inject Google Cloud test repo plugin if using staging or unstable repos.
  # This is used to test new package releases in images.
  if repo != 'stable':
    logging.info('Adding Google Cloud test repos plugin for bootstrapvz.')
    repo_plugin_dir = '/files/google_cloud_test_repos'
    bvz_plugins = os.path.join(BVZ_DIR, 'bootstrapvz', 'plugins')
    shutil.move(repo_plugin_dir, bvz_plugins)

    with open(bvz_manifest_file, 'r+') as manifest_file:
      manifest_data = yaml.load(manifest_file)
      manifest_plugins = manifest_data['plugins']
      manifest_plugins['google_cloud_test_repos'] = {repo: True}
      manifest_yaml = yaml.dump(manifest_data, default_flow_style=False)
      manifest_file.write(manifest_yaml)

  # Run bootstrap_vz build.
  cmd = [bvz_bin, '--debug', bvz_manifest_file]
  logging.info('Starting build in %s with params: %s' % (BVZ_DIR, str(cmd)))
  utils.Execute(cmd, cwd=BVZ_DIR)

  # Upload tar.
  image_tar_gz = '/target/disk.tar.gz'
  if os.path.exists(image_tar_gz):
    logging.info('Saving %s to %s' % (image_tar_gz, image_dest))
    utils.UploadFile(image_tar_gz, image_dest)

  # Create and upload the synopsis of the image.
  logging.info('Creating image synopsis.')
  synopsis = {}
  packages = collections.OrderedDict()
  _, output = utils.Execute(['dpkg-query', '-W'], capture_output=True)
  for line in output.split('\n')[:-1]:  # Last line is an empty line.
    parts = line.split()
    packages[parts[0]] = parts[1]
  synopsis['installed_packages'] = packages
  with open('/tmp/synopsis.json', 'w') as f:
    f.write(json.dumps(synopsis))
  logging.info('Uploading image synopsis.')
  synopsis_dest = os.path.join(outs_path, 'synopsis.json')
  utils.UploadFile('/tmp/synopsis.json', synopsis_dest)

  # Create and upload metadata of the image and packages
  logging.info('Creating image metadata.')
  image = {
      "family": "debian-9",
      "version": "stretch",
      "location": image_dest,
      "build_date,": datetime.now(timezone.utc),
      "build_repo": repo,
      "packages": []
  }
  # Read list of guest package
  with open("guest_package") as f:
    guest_packages = f.read().splitlines()

  for package in guest_packages:
    cmd = ("dpkg-query -W --showformat '${Package}\n${Version}\n${Git}' "
           "%s") % package
    _, stdout = utils.Excute(cmd, capture_output=True)
    package, version, git = stdout.decode('utf-8').split(':', 2)
    metadata = {
        "name": package,
        "version": version,
        "commit_hash": git,
    }
    image["packages"].append(metadata)

    # Write image metadata to a file
    with open('/tmp/metadata.json', 'w') as f:
      f.write(json.dumps(image))

    logging.info('Uploading image metadata.')
    utils.UploadFile('/tmp/metadata.json', image_meta)


if __name__ == '__main__':
  try:
    main()
    logging.success('Debian build was successful!')
  except Exception as e:
    logging.error('Debian build failed: %s' % str(e))
