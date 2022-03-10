#!/usr/bin/env python3


import logging
import re
import subprocess

import utils
from utils.common import _GetMetadataParam


def run(cmd, capture_output=True, check=True, encoding='utf-8'):
  logging.info('Run: %s', cmd)
  return subprocess.run(cmd.split(), capture_output=capture_output,
                        check=check, encoding=encoding)


def get_mount_disk(image):
  devname = _GetMetadataParam('disks/1/device-name', raise_on_not_found=True)
  devicepath = f'/dev/disk/by-id/google-{devname}'
  gpt = get_part_type(devicepath) == 'gpt'

  # This assumes that, for UEFI systems:
  # 1. partition 1 is the EFI system partition.
  # 2. partition 2 is the root mount for the installed system.
  #
  # Except on debian 10, which has out-of-order partitions.
  if gpt and 'debian-10' not in image and 'debian-11' not in image:
    return f'{devicepath}-part2'
  else:
    return f'{devicepath}-part1'


def get_part_type(device):
  ret = run(f'blkid -s PTTYPE -o value {device}')
  return ret.stdout.strip()


def get_distro_from_image(image):
  el_distros = ('centos', 'rhel', 'almalinux', 'rockylinux')
  if any([x in image for x in el_distros]):
    return 'enterprise_linux'
  elif 'debian' in image:
    return 'debian'
  else:
    return None


def main():
  image = utils.GetMetadataAttribute('image', raise_on_not_found=True)
  package = utils.GetMetadataAttribute('gcs_package_path',
                                       raise_on_not_found=True)
  package_name = package.split('/')[-1]

  mount_disk = get_mount_disk(image)
  logging.info('Mount device %s at /mnt', mount_disk)
  run(f'mount {mount_disk} /mnt')

  # The rpm utility requires /dev/random to initialize GnuTLS
  logging.info('Mount dev filesystem in chroot')
  run('mount -o bind /dev /mnt/dev')

  utils.DownloadFile(package, f'/mnt/tmp/{package_name}')

  distribution = get_distro_from_image(image)
  if distribution == 'debian':
    util = 'apt-get'
  elif distribution == 'enterprise_linux':
    util = 'yum'
  else:
    logging.error('Unknown Linux distribution.')
    return

  logging.info('Installing package %s', package_name)
  run(f'chroot /mnt {util} install -y /tmp/{package_name}')
  if distribution == 'enterprise_linux':
    # (google-guest-agent)-20210723.01
    m = re.search(r'(.+)-[0-9]{8}\.[0-9]{2}.*', package_name)
    if not m:
      logging.error('unknown package name, cant relabel')
      return
    package_short_name = m.group(1)

    out = run(f'chroot /mnt rpm -ql {package_short_name}')
    with open('/mnt/tmp/relabel', 'w') as fp:
      fp.write(out.stdout)
    run('chroot /mnt /sbin/setfiles -v -F -f /tmp/relabel '
        '/etc/selinux/targeted/contexts/files/file_contexts')

  # Best effort to unmount prior to shutdown.
  run('sync', check=False)
  run('umount /mnt/dev', check=False)
  run('umount /mnt', check=False)

  logging.success('Package %s installed successfully', package_name)


if __name__ == '__main__':
  try:
    main()
  except subprocess.CalledProcessError as e:
    logging.info('stdout: %s', e.stdout)
    logging.info('stderr: %s', e.stderr)
    logging.error('failed to execute cmd: %s', e)
  except Exception as e:
    logging.error('%s', e)
