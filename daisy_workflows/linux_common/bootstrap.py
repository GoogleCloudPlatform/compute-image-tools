#!/usr/bin/env python2
# Copyright 2018 Google Inc. All Rights Reserved.
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

"""Bootstrapper for running a VM script.

Args:
  files_gcs_dir: The Cloud Storage location containing the files.
    This dir will be used to run the 'script' requested by Metadata.
  script: The main script to be run
  prefix: a string prefix for outputing status
"""
import ast
import json
import logging
import os
import subprocess
import urllib2


DIR = '/files'
TOKEN = None


def GetAccessToken():
  url = '%(metadata)s/v1/instance/service-accounts/default/token' % {
      'metadata': 'http://metadata.google.internal/computeMetadata',
  }
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  # converts the stringified dictionary of the response to a dictionary
  response = ast.literal_eval(urllib2.urlopen(request).read())
  return '%s %s' % (response[u'token_type'], response[u'access_token'])


def GetBucketContent(bucket):
  url = '%(storage)s/v1/b/%(bucket_name)s/o' % {
      'storage': 'https://www.googleapis.com/storage',
      'bucket_name': bucket,
  }
  print ("Bucket listing: %s" % url)
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  request.add_unredirected_header('Authorization', TOKEN)
  content = json.load(urllib2.urlopen(request))
  return [i['name'] for i in content['items']]


def SaveBucketFile(bucket, bucket_file, dest_filepath):
  url = 'https://storage.googleapis.com/%s/%s' % (bucket, bucket_file)
  print ("Bucket save: %s => %s" % (url, dest_filepath))
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  request.add_unredirected_header('Authorization', TOKEN)
  content = urllib2.urlopen(request).read()
  f = open(dest_filepath, 'w')
  f.write(content)
  f.close()


def GetMetadataAttribute(attribute):
  url = '%(metadata)s/v1/instance/attributes/%(attribute_name)s' % {
      'metadata': 'http://metadata.google.internal/computeMetadata',
      'attribute_name': attribute,
  }
  request = urllib2.Request(url)
  request.add_unredirected_header('Metadata-Flavor', 'Google')
  return urllib2.urlopen(request).read()


def DebianInstallGoogleApiPythonClient(prefix):
  logging.info('%sStatus: Installing google-api-python-client', prefix)
  subprocess.check_call(['apt-get', 'update'])
  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  cmd = ['apt-get', '-q', '-y', 'install', 'python-pip']
  subprocess.Popen(cmd, env=env).communicate()

  cmd = ['pip', 'install', '--upgrade', 'google-api-python-client']
  subprocess.check_call(cmd)


def Bootstrap():
  """Get files, run."""
  try:
    global TOKEN
    prefix = GetMetadataAttribute('prefix')
    status = prefix + 'Status'
    logging.info('%s: Starting bootstrap.py.', status)

    # Optional flag
    try:
      if GetMetadataAttribute('debian_install_google_api_python_client'):
        DebianInstallGoogleApiPythonClient(prefix)
    except urllib2.HTTPError:
      pass

    TOKEN = GetAccessToken()
    gcs_dir = GetMetadataAttribute('files_gcs_dir')
    script = GetMetadataAttribute('script')
    full_script = os.path.join(DIR, script)
    subprocess.check_call(['mkdir', DIR])

    # Copies all files from bucket's gcs_dir to DIR
    path_stripped = gcs_dir[len('gs://'):]
    token = path_stripped.find('/')

    # skip leading slash on bucket_dir
    bucket, bucket_dir = path_stripped[:token], path_stripped[token + 1:]

    def BelongsToBucketDir(filename):
      return filename.startswith(bucket_dir)

    bucket_files = filter(BelongsToBucketDir, GetBucketContent(bucket))
    for f in bucket_files:
      dest_filepath = f.replace(bucket_dir, DIR)
      SaveBucketFile(bucket, f, dest_filepath)

    logging.info('%s: Making script %s executable.', status, full_script)
    subprocess.check_call(['chmod', '+x', script], cwd=DIR)
    logging.info('%s: Running %s.', status, full_script)
    subprocess.check_call([full_script], cwd=DIR)
  except Exception as e:
    fail = prefix + 'Failed'
    print('%s: error: %s' % (fail, str(e)))


if __name__ == '__main__':
  logging.basicConfig(level=logging.DEBUG)
  Bootstrap()
