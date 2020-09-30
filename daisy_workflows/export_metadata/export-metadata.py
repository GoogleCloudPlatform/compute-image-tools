#!/usr/bin/env python3

"""Export the image metadata on a GCE VM.

Parameters retrieved from instance metadata:

metadata_dest: The Cloud Storage destination for the resultant image metadata.
image_id: The resource id of the image.
image_name: The name of the image in GCP.
image_family: The family that image belongs.
distribution: Use image distribution. Must be one of [enterprise_linux,
  debian, centos].
uefi: boolean Whether using UEFI to boot OS.
"""

import datetime
import json
import logging
import subprocess
import sys
import tempfile

import utils


def main():
  # Get parameters from instance metadata.
  metadata_dest = utils.GetMetadataAttribute('metadata_dest',
                                             raise_on_not_found=True)
  image_id = utils.GetMetadataAttribute('image_id',
                                        raise_on_not_found=True)
  image_name = utils.GetMetadataAttribute('image_name',
                                          raise_on_not_found=True)
  image_family = utils.GetMetadataAttribute('image_family',
                                            raise_on_not_found=True)
  distribution = utils.GetMetadataAttribute('distribution',
                                            raise_on_not_found=True)
  uefi = utils.GetMetadataAttribute('uefi', 'false').lower() == 'true'

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
    utils.UploadFile(f.name, metadata_dest)
  except ValueError as e:
    logging.exception('ExportFailed: Failed uploading metadata file %s', e)
    sys.exit(1)

  logging.info('ExportSuccess: Export metadata was successful!')


if __name__ == '__main__':
  main()
