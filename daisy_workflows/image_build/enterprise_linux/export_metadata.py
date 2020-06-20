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
import time
import uuid
from datetime import datetime
from datetime import timezone

import utils


def main():
  # Get Parameters.
  repo = utils.GetMetadataAttribute('google_cloud_repo',
                                    raise_on_not_found=True).strip()
  el_release_name = utils.GetMetadataAttribute('el_release',
                                               raise_on_not_found=True)
  build_date = utils.GetMetadataAttribute('build_date',
                                          raise_on_not_found=True)
  destination_prefix = utils.GetMetadataAttribute('destination_prefix',
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
  image = {
      "id": uuid.uuid4(),
      "family": el_release_name,
      "name": el_release_name + build_date,
      "version": build_date,
      "location": "gs://gce-image-archive/centos-uefi/centos-7-v${build_date}",
      "release_date": datetime.now(timezone.utc),
      "state": "Active",
      "environment": "prod",
      "packages": []
  }

  # Read list of guest package
  with open("guest_package") as f:
    guest_packages = f.read().splitlines()

  for package in guest_packages:
    # The URL is a github repo link which don't contain commit hash
    cmd = "rpm -q --queryformat='%{NAME} %{VERSION} %{RELEASE} %{URL}\n'" \
          + package
    code, stdout = utils.Excute(cmd, capture_output=True)
    if code == 0:
      splits = stdout.decode('utf-8').split('\t\b')
      package_name = splits[0]
      package_version = splits[1] + "-" + splits[2]
      package_release_date = package_version[0:package_version.index(".")]
      metadata = {
          "id": uuid.uuid4(),
          "name": package_name,
          "version": package_version,
          # For el, we don't have commit hash
          "commit_hash": "",
          "release_date": package_release_date,
          "stage": repo
      }
      image["packages"].append(metadata)

      with open('/tmp/metadata.json', 'w') as f:
        f.write(json.dumps(image))

      logging.info('Uploading image metadata.')
      metadata_dest = os.path.join(destination_prefix, 'metadata.json')
      utils.UploadFile('/tmp/metadata.json', metadata_dest)


if __name__ == '__main__':
  try:
    main()
    logging.success('Export EL metadata was successful!')
  except Exception as e:
    logging.error('Export EL metadata failed: %s' % str(e))
