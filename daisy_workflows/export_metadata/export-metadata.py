#!/usr/bin/env python3

"""Export the image metadata on a GCE VM.

Parameters retrieved from instance metadata:

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
  image_id = utils.GetMetadataAttribute('image_id')
  image_name = utils.GetMetadataAttribute('image_name')
  image_family = utils.GetMetadataAttribute('image_family')
  distribution = utils.GetMetadataAttribute('distribution',
                                            raise_on_not_found=True)
  uefi = utils.GetMetadataAttribute('uefi', 'false').lower() == 'true'
  outs_path = utils.GetMetadataAttribute('daisy-outs-path')

  logging.info('Creating upload metadata of the image and packages.')

  utc_time = datetime.datetime.now(datetime.timezone.utc)
  image_version = image_name.rsplit("v")[-1]
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
    mount_disk = unmounted_root_fs()
    if mount_disk is None:
      logging.error('Could not find scanned disk root fs')
      return

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
                  '${Package}\n\n${Version}\n${Git}']
  elif distribution == 'enterprise_linux':
      cmd_prefix = ['chroot', '/mnt', 'rpm', '-q', '--queryformat',
                    '%{NAME}\n%{EPOCH}\n%{VERSION}-%{RELEASE}\n%{VCS}']
  else:
    logging.error('Unknown Linux distribution.')
    return

  for package in guest_packages:
    try:
      process = subprocess.run(cmd_prefix + [package], capture_output=True,
                               check=True)
    except subprocess.CalledProcessError as e:
      logging.info('failed to execute cmd: %s stdout: %s stderr: %s', e,
                   e.stdout, e.stderr)
      continue

    stdout = process.stdout.decode()

    try:
      package, epoch, version, commit_hash = stdout.split('\n', 3)
    except ValueError:
      logging.info('command result was malformed: %s', stdout)
      continue

    md = make_pkg_metadata(package, version, epoch, commit_hash)
    image['packages'].append(md)

  # Write image metadata to a file.
  with tempfile.NamedTemporaryFile(mode='w', dir='/tmp', delete=False) as f:
    f.write(json.dumps(image))

  # We upload the result to the daisy outs path as well, to aid in
  # troubleshooting.
  logging.info('Uploading image metadata to daisy outs path.')
  try:
    utils.UploadFile(f.name, outs_path + "/metadata.json")
  except Exception as e:
    logging.error('Failed uploading metadata file %s', e)
    return

  logging.success('Export metadata was successful!')


def make_pkg_metadata(name, version, epoch, commit_hash):
  # The epoch field is only present in certain packages and will return
  # '(none)' otherwise.
  if 'none' in epoch:
    epoch = ''

  # Match debian style version which includes epoch.
  if epoch:
    version = '%s:%s' % (epoch, version)

  return {'name': name, 'version': version, 'commit_hash': commit_hash}


def root_fs(block_device):
    # Given a block device returns the partition's device node of a root fs
    # partition. It looks for a partition with the root fs guuid. Returns
    # None if no root fs is found.
    root_guuid = '0fc63daf-8483-4772-8e79-3d69d8477de4'
    process = subprocess.run(['lsblk', block_device, '-lno', 'NAME'],
                             capture_output=True, check=True)

    for ln in process.stdout.decode().split():
        part = '/dev/' + ln
        if part == block_device:
            continue
        process = subprocess.run(['blkid', '-p', part],
                                 capture_output=True, check=True)
        result = process.stdout.decode().replace(part + ': ', '')
        tokens = result.replace('"', '').split(' ')
        data = {i.split('=')[0]: i.split('=')[1] for i in tokens}

        entry_type = data.get('PART_ENTRY_TYPE', None)
        if entry_type == root_guuid:
          return part

        label = data.get('LABEL', None)
        if label == 'root':
          return part

    return None


def unmounted_root_fs():
    # Searches for the unmounted root fs in the system, we know there'll be 2
    # block devices the worker's disk and the scanned image/disk so we test
    # for /dev/sda and /dev/sdb.
    process = subprocess.run(['mount'], capture_output=True, check=True)
    mounted_parts = process.stdout.decode().split()

    for curr in ['/dev/sda', '/dev/sdb']:
        device_node = root_fs(curr)
        if device_node is None:
            continue
        is_mounted = False
        for ln in mounted_parts:
            if device_node in ln:
                is_mounted = True
                break
        if not is_mounted:
            return device_node

    return None


if __name__ == '__main__':
  main()
