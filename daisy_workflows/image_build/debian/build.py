#!/usr/bin/python
"""Build the Debian image on a GCE VM.

Parameters (retrieved from instance metadata):

bootstrap-vz-manifest: The version of bootstrap-vz to retrieve and use.
bootstrap-vz-version: The version of bootstrap-vz to retrieve and use.
google-cloud-repo: The repo to use to build Debian. Must be one of
  ['stable' (default), 'unstable', 'staging'].
image-dest: The Cloud Storage destination for the resultant image.
license-id: The Compute Engine license-id for this release of Debian.
release: The Debian release being built.


debian-public-git-urls: The public git repositories to use to build Debian.
debian-internal-git-urls: The internal git repositories to use to build Debian.
custom-packages: If set, downloads all *.deb packages from Cloud Storage bucket
                 and installs them.
build-backports: Build image with Backports kernel
image-dest: Cloud storage path to upload the image to.

The debian-*-git-urls parameters are any bootstrap-vz tree git can clone from
GCE, optionally followed by ";branch" to select something other than the
repository's default.
"""

import collections
import glob
import json
import logging
import os
import shutil
import tarfile
import urllib
import zipfile

import utils

utils.AptGetInstall(
    ['git', 'python-pip', 'qemu-utils', 'parted', 'kpartx', 'debootstrap',
     'python-yaml'])
utils.PipInstall(['termcolor', 'fysom', 'jsonschema', 'docopt', 'functools32'])

import yaml

BVZ_DIR = '/bvz'
BVZ_MANIFEST = ''  # Populated at runtime.
REPOS = ['stable', 'unstable', 'staging']


def main():
  # Get Parameters.
  BVZ_MANIFEST = utils.GetMetadataParam(
      'bootstrap-vz-manifest', raise_on_not_found=True)
  bvz_version = utils.GetMetadataParam(
      'bootstrap-vz-version', raise_on_not_found=True)
  build_files_gcs_dir = utils.GetMetadataParam(
      'build-files-gcs-dir', raise_on_not_found=True)
  repo = utils.GetMetadataParam('google-cloud-repo', raise_on_not_found=True)
  image_dest = utils.GetMetadataParam('image-dest', raise_on_not_found=True)
  license_id = utils.GetMetadataParam('license-id', raise_on_not_found=True)
  if repo not in REPOS:
    raise ValueError(
        'Metadata "google-cloud-repo" must be one of %s.' % REPOS)
  release = utils.GetMetadataParam('release', raise_on_not_found=True)

  logging.info('Debian Builder')
  logging.info('==============')
  logging.info('Image Path: %s', image_dest)
  logging.info('Bootstrap_vz manifest: %s', BVZ_MANIFEST)
  logging.info('Bootstrap_vz version: %s', bvz_version)
  logging.info('Google Cloud repo: %s', repo)
  logging.info('Debian Builder Sources: %s', build_files_gcs_dir)

  # Download and setup bootstrap_vz.
  bvz_url = 'https://github.com/andsens/bootstrap-vz/archive/%s.zip'
  bvz_url %= bvz_version
  bvz_zip_dir = 'bvz_zip'
  logging.info('Downloading bootstrap-vz')
  urllib.urlretrieve(bvz_url, 'bvz.zip')
  with zipfile.ZipFile('bvz.zip', 'r') as z:
    z.extractall(bvz_zip_dir)
  logging.info('Downloaded and extracted %s to %s', bvz_url, 'bvz_zip')
  bvz_zip_contents = [d for d in os.listdir(bvz_zip_dir)]
  bvz_zip_subdir = os.path.join(bvz_zip_dir, bvz_zip_contents[0])
  utils.Execute(['mv', bvz_zip_subdir, BVZ_DIR])
  logging.info('Moved bootstrap_vz from %s to %s.', bvz_zip_subdir, BVZ_DIR)
  bvz_bin = os.path.join(BVZ_DIR, 'bootstrap-vz')
  utils.MakeExecutable(bvz_bin)
  logging.info('Made %s executable.', bvz_bin)

  # Run bootstrap_vz build.
  cmd = [bvz_bin, '--debug', os.path.join(BVZ_DIR, 'manifests', BVZ_MANIFEST)]
  logging.info('Starting build in %s with params: %s', BVZ_DIR, str(cmd))
  utils.Execute(cmd, cwd=BVZ_DIR)

  # Setup tmpfs.
  tmpfs = '/mnt/tmpfs'
  os.makedirs(tmpfs)
  utils.Execute(['mount', '-t', 'tmpfs', '-o', 'size=20g', 'tmpfs', tmpfs])

  # Create license manifest.
  license_manifest = os.path.join(tmpfs, 'manifest.json')
  logging.info('Creating license manifest for %s', license_id)
  manifest = '{"licenses": ["%s"]}' % license_id
  with open(license_manifest, 'w') as manifest_file:
    manifest_file.write(manifest)

  # Extract raw image.
  image = '/target/disk.tar.gz'
  logging.info('Creating licensed tar for %s', image)
  with tarfile.open(image, 'r:gz') as tar:
    tar.extractall(tmpfs)

  # Create tar with license manifest included.
  disk_raw_tar = os.path.join(tmpfs, os.path.basename(image))
  with tarfile.open(disk_raw_tar, 'w:gz') as tar:
    tar_info = tarfile.TarInfo(name='disk.raw')
    tar_info.type = tarfile.GNUTYPE_SPARSE
    tar.add(license_manifest, arcname='manifest.json')
    tar.add(os.path.join(tmpfs, 'disk.raw'), arcname='disk.raw')

  # Upload tar.
  disk_raw_tar_dest = os.path.join(image_dest, 'disk.tar.gz')
  logging.info('Saving %s to %s', disk_raw_tar, disk_raw_tar_dest)
  utils.Gsutil(['cp', disk_raw_tar, disk_raw_tar_dest])

  # Create and upload the synopsis of the image.
  logging.info('Creating image synopsis.')
  synopsis = {}
  packages = collections.OrderedDict()
  _, output, _ = utils.Execute(['dpkg-query', '-W'], capture_output=True)
  for line in output.split('\n')[:-1]:  # Last line is an empty line.
    parts = line.split()
    packages[parts[0]] = parts[1]
  synopsis['installed_packages'] = packages
  with open('/tmp/synopsis.json', 'w') as f:
    f.write(json.dumps(synopsis))
  logging.info('Uploading image synopsis.')
  synopsis_dest = os.path.join(image_dest, 'synopsis.json')
  utils.Gsutil(['cp', '/tmp/synopsis.json', synopsis_dest])

if __name__ == '__main__':
  utils.RunScript(main)
