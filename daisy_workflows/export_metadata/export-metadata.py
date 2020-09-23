#!/usr/bin/env python3

"""Export the image metadata on a GCE VM.

Parameters retrieved from instance metadata:

google_cloud_repo: The repo to use to build image. Must be one of
  ['stable' (default), 'unstable', 'staging'].
image_tar_location: The Cloud Storage destination for the resultant image.
metadata_dest: The Cloud Storage destination for the resultant image metadata.
image_family: The family that image belongs.
distribution: Use image distribution. Must be one of [enterprise_linux,
  debian, centos].
uefi: boolean Whether using UEFI to boot OS.
"""

import datetime
import json
import logging
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request

from google.cloud import exceptions
from google.cloud import storage

METADATA_ENDPOINT = 'http://metadata.google.internal/computeMetadata/v1/instance/attributes/?recursive=true'


def GetMetadataAttribute():
  """Get attribute from metadata server.

  Returns:
    dictionary, the instance attribute metadata.
  """
  try:
    request = urllib.request.Request(METADATA_ENDPOINT)
    headers = {'Metadata-Flavor': 'Google'}
    for key, value in headers.items():
      request.add_unredirected_header(key, value)
    return json.loads(urllib.request.urlopen(request).read().decode())
  except urllib.error.HTTPError:
    raise ValueError('Metadata key not found')


def UploadFile(source_file, gcs_dest_file):
  """Uploads a file to GCS.

  Expects a local source file and a destination bucket and GCS path.

  Args:
    source_file: string, the path of a source file to upload.
    gcs_dest_file: string, the path to the resulting file in GCS.

  Raises:
    ValueError: The error occurred when gcs_dest_file is invalid bucket.
  """

  bucket = r'(?P<bucket>[a-z0-9][-_.a-z0-9]*[a-z0-9])'
  obj = r'(?P<obj>[^\*\?]+)'
  prefix = r'gs://'
  gs_regex = re.compile(r'{prefix}{bucket}/{obj}'
                        .format(prefix=prefix, bucket=bucket, obj=obj))
  match = gs_regex.match(gcs_dest_file)
  if not match:
    raise ValueError('Destination path %s is invalid.'% gcs_dest_file)
  client = storage.Client()
  bucket = client.get_bucket(match.group('bucket'))
  blob = bucket.blob(match.group('obj'))
  try:
    blob.upload_from_filename(source_file)
  except exceptions.from_http_status as e:
    raise ValueError('Upload to bucket %s failed.' % gcs_dest_file)


def main():
  # Get parameters from instance metadata.
  metadata = GetMetadataAttribute()
  metadata_dest = metadata.get('metadata_dest')
  image_id = metadata.get('image_id')
  image_name = metadata.get('image_name')
  image_family = metadata.get('image_family')
  distribution = metadata.get('distribution')
  uefi = metadata.get('uefi', '').lower() == 'true'

  logging.info('Creating upload metadata of the image and packages.')

  utc_time = datetime.datetime.now(datetime.timezone.utc)
  image_version = utc_time.strftime('%Y%m%d')
  build_date = utc_time.astimezone().isoformat(),
  image = {
      'id': image_id,
      'name': image_name,
      'family': image_family,
      'version': image_version,
      'build_date,': build_date,
      'packages': [],
  }
  # All the guest environment packages maintained by guest-os team.
  guest_packages = [
      'google-cloud-packages-archive-keyring',
      'google-compute-engine',
      'google-compute-engine-oslogin',
      'google-guest-agent',
      'google-osconfig-agent',
      'gce-disk-expand',
  ]

  # This assumes that:
  # 1. /dev/sdb1 is the EFI system partition.
  # 2. /dev/sdb2 is the root mount for the installed system.
  if uefi:
    mount_disk = '/dev/sdb2'
  else:
    mount_disk = '/dev/sdb1'
  subprocess.run(['mount', mount_disk, '/mnt'], check=False)
  logging.info('Mount %s device to /mnt', mount_disk)

  if distribution == 'enterprise_linux':
    # chroot prevents access to /dev/random and /dev/urandom (as designed).
    # The rpm required those random bits to initialize GnuTLS otherwise
    # error: Failed to initialize NSS library.
    subprocess.run(['mount', '-o', 'bind', '/dev', '/mnt/dev'], check=False)

  has_commit_hash = True
  if distribution == 'debian':
    cmd_prefix = ['chroot', '/mnt', 'dpkg-query', '-W', '--showformat',
                  '${Package}\n${Version}\n${Git}']
  elif distribution == 'enterprise_linux':
    if 'centos-6' in image_family or 'rhel-6' in image_family:
      # centos-6 and rhel-6 doesn't support vcs tag
      cmd_prefix = ['chroot', '/mnt', 'rpm', '-q', '--queryformat',
                    '%{NAME}\n%{VERSION}-%{RELEASE}']
      has_commit_hash = False
    else:
      cmd_prefix = ['chroot', '/mnt', 'rpm', '-q', '--queryformat',
                    '%{NAME}\n%{VERSION}-%{RELEASE}\n%{VCS}']
  else:
    logging.error('Unknown Linux distribution.')
    return Exception

  version, commit_hash = '', ''
  for package in guest_packages:
    cmd = cmd_prefix + [package]
    try:
      stdout = subprocess.run(cmd, stdout=subprocess.PIPE, check=True).stdout
      stdout = stdout.decode()
      logging.info('Package metadata is %s', stdout)
    except subprocess.CalledProcessError as e:
      logging.exception('Fail to execute cmd. %s', e)
      continue
    if has_commit_hash:
      package, version, commit_hash = stdout.split('\n', 2)
    else:
      package, version = stdout.split('\n', 1)
    package_metadata = {
        'name': package,
        'version': version,
        'commit_hash': commit_hash,
    }
    image['packages'].append(package_metadata)

  # Write image metadata to a file.
  with tempfile.NamedTemporaryFile(mode='w', dir='/tmp', delete=False) as f:
    f.write(json.dumps(image))

  logging.info('Uploading image metadata.')
  try:
    UploadFile(f.name, metadata_dest)
  except ValueError as e:
    logging.exception('ExportFailed: Failed uploading metadata file %s', e)
    sys.exit(1)

  logging.info('ExportSuccess: Export metadata was successful!')


if __name__ == '__main__':
  main()
