#!/usr/bin/python
"""Bootstrapper for running a VM script.

Args:
build-files-gcs-dir: The Cloud Storage location containing the build files.
  This dir of build files must contain a build.py containing the build logic.
"""
import base64
import logging
import os
import subprocess
import urllib2
import zipfile


def GetMetadataAttribute(attribute):
  url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % attribute
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def Bootstrap():
  """Get build files, run build, poweroff."""
  try:
    logging.info('Starting bootstrap.py.')
    build_gcs_dir = GetMetadataAttribute('build-files-gcs-dir')
    build_script = GetMetadataAttribute('build-script')
    build_dir = '/build_files'
    full_build_script = os.path.join(build_dir, build_script)
    subprocess.call(['mkdir', build_dir])
    subprocess.call(
        ['gsutil', 'cp', '-r', os.path.join(build_gcs_dir, '*'), build_dir])
    logging.info('Making build script %s executable.', full_build_script)
    subprocess.call(['chmod', '+x', build_script], cwd=build_dir)
    logging.info('Running %s.', full_build_script)
    subprocess.call([full_build_script], cwd=build_dir)
  finally:
    os.system('sync')
    os.system('shutdown now -h')

if __name__ == '__main__':
  Bootstrap()
