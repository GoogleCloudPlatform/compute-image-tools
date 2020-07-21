#!/usr/bin/env python3
# Copyright 2020 Google Inc. All Rights Reserved.
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

"""Export image metadata on a GCE VM.

Parameters (retrieved from instance metadata):
google_cloud_repo: The package repo to use. Can be stable (default), staging,
  or unstable.
el_release: rhel6, rhel7, centos6, centos7, oraclelinux6, or oraclelinux7
image_dest: The Cloud Storage destination for the resultant image.
destination_prefix: The CS path to export image metadata to
"""

from datetime import datetime, timezone
import json
import logging
import os

import utils


def main():
  # Get Parameters.
  repo = utils.GetMetadataAttribute('google_cloud_repo',
                                    raise_on_not_found=True)
  el_release_name = utils.GetMetadataAttribute('el_release',
                                               raise_on_not_found=True)
  destination = utils.GetMetadataAttribute('destination',
                                           raise_on_not_found=True)
  uefi = utils.GetMetadataAttribute('rhel_uefi') == 'true'

  # Mount the installer disk.
  if uefi:
    utils.Execute(['mount', '/dev/sdb2', '/mnt'])
  else:
    utils.Execute(['mount', '/dev/sdb1', '/mnt'])

  logging.info('Installer root: %s' % os.listdir('/mnt'))

  # Create and upload metadata of the image and packages
  logging.info('Creating image metadata.')
  build_date = datetime.today().strftime('%Y%m%d')
  image = {
      "family": el_release_name,
      "version": build_date,
      "location": destination,
      "release_date": datetime.now(timezone.utc),
      "build_repo": repo,
      "packages": []
  }

  # Read list of guest package
  with open("guest_package") as f:
    guest_packages = f.read().splitlines()

  for package in guest_packages:
    cmd = ("rpm -q "
          "--queryformat='%%{NAME}\n%%{VERSION}\n%%{RELEASE}\n%%{Vcs}' "
          "%s") % package
    _, stdout = utils.Excute(cmd, capture_output=True)
    package, version, release, vcs = stdout.decode('utf-8').split('\n', 3)
    package_metadata = {
        "name": package,
        "version": version + "-" + release,
        "commit_hash": vcs
    }
    image["packages"].append(package_metadata)

    # Write image metadata to a file
    with open('/tmp/metadata.json', 'w') as f:
      f.write(json.dumps(image))

    logging.info('Uploading image metadata.')
    utils.UploadFile('/tmp/metadata.json', destination)


if __name__ == '__main__':
  try:
    main()
    logging.success('Export EL metadata was successful!')
  except Exception as e:
    logging.error('Export EL metadata failed: %s' % str(e))
