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
import tempfile

import utils


def main():
  # Get parameters from instance metadata.
  metadata_dest = utils.GetMetadataAttribute('metadata_dest',
                                             raise_on_not_found=True)
  image_id = utils.GetMetadataAttribute('image_id')
  image_name = utils.GetMetadataAttribute('image_name')
  image_family = utils.GetMetadataAttribute('image_family')
  distribution = utils.GetMetadataAttribute('distribution',
                                            raise_on_not_found=True)
  uefi = utils.GetMetadataAttribute('uefi', 'false').lower() == 'true'
  outs_path = utils.GetMetadataAttribute('daisy-outs-path')

  logging.info('Creating upload metadata of the image and packages.')

  utc_time = datetime.datetime.now(datetime.timezone.utc)
  image_version = utc_time.strftime('%Y%m%d')
  publish_date = utc_time.astimezone().isoformat()
  image = {
      'id': image_id,
      'name': image_name,
      'family': image_family,
      'version': image_version,
      'publish_date': publish_date,
      'packages': [],
  }

  # All the guest environment packages maintained by guest-os team.
  guest_packages = [
      'google-compute-engine',
      'google-compute-engine-oslogin',
      'google-guest-agent',
      'google-osconfig-agent',
      'gce-disk-expand',
  ]

  # This assumes that:
  # 1. /dev/sdb1 is the EFI system partition.
  # 2. /dev/sdb2 is the root mount for the installed system.
  # Except for debian 10, which has out-of-order partitions.
  if uefi and 'debian-10' not in image_family:
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

  if distribution == 'debian':
    #  This package is debian-only.
    guest_packages.append('google-cloud-packages-archive-keyring')
    cmd_prefix = ['chroot', '/mnt', 'dpkg-query', '-W', '--showformat',
                  '${Package}\n${Version}\n${Git}']
  elif distribution == 'enterprise_linux':
      cmd_prefix = ['chroot', '/mnt', 'rpm', '-q', '--queryformat',
                    '%{NAME}\n%{EPOCH}\n%{VERSION}-%{RELEASE}\n%{VCS}']
  else:
    logging.error('Unknown Linux distribution.')
    return

  version = ''
  commit_hash = ''
  for package in guest_packages:
    cmd = cmd_prefix + [package]
    try:
      stdout = subprocess.run(cmd, stdout=subprocess.PIPE, check=True).stdout
      stdout = stdout.decode()
      logging.info('Package metadata is %s', stdout)
    except subprocess.CalledProcessError as e:
      logging.error('Fail to execute cmd. %s', e)
      return

    package, epoch, version, commit_hash = stdout.split('\n', 3)
    if 'none' in epoch:
      # The epoch field is only present in certain packages and will return
      # '(none)' otherwise.
      epoch = ''
    package_metadata = {
        'name': package,
        # Match debian version formatting of epoch, if present.
        'version': '{}:{}'.format(epoch, version) if epoch else version,
        'commit_hash': commit_hash,
    }
    image['packages'].append(package_metadata)

  # Write image metadata to a file.
  with tempfile.NamedTemporaryFile(mode='w', dir='/tmp', delete=False) as f:
    f.write(json.dumps(image))

  # We upload the result to the daisy outs path as well, to aid in
  # troubleshooting.
  logging.info('Uploading image metadata to daisy outs path.')
  try:
    utils.UploadFile(f.name, outs_path + "/metadata.json")
  except ValueError as e:
    logging.error('Failed uploading metadata file %s', e)
    return

  logging.info('Uploading image metadata to destination.')
  try:
    utils.UploadFile(f.name, metadata_dest)
  except ValueError as e:
    logging.error('Failed uploading metadata file %s', e)
    return

  logging.success('Export metadata was successful!')


if __name__ == '__main__':
  main()
