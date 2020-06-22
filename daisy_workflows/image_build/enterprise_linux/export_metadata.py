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

import json
import logging
import os
from datetime import datetime
from datetime import timezone

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
      "name": el_release_name + build_date,
      "version": build_date,
      "location": destination_prefix,
      "release_date": datetime.now(timezone.utc),
      "state": "Active",
      "packages": []
  }

  # Read list of guest package
  with open("guest_package") as f:
    guest_packages = f.read().splitlines()

  for package in guest_packages:
    # The URL is a github repo link which don't contain commit hash
    cmd = "rpm -q --queryformat='%{NAME}\n%{VERSION}\n%{RELEASE}\n%{URL}'" \
          + package
    code, stdout = utils.Excute(cmd, capture_output=True)
    if code == 0:
      splits = stdout.decode('utf-8').split('\n')
      package_name = splits[0]
      package_version = splits[1] + "-" + splits[2]
      package_url = splits[3]
      package_release_date = package_version[0:package_version.index(".")]
      package_metadata = {
          "name": package_name,
          "version": package_version,
          # For el, we don't have commit hash
          "commit_hash": package_url,
          "release_date": package_release_date,
          "stage": repo
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
