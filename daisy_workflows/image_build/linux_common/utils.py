#!/usr/bin/python
# Copyright 2017 Google Inc. All Rights Reserved.
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

"""Utility functions for all VM scripts."""

import glob
import json
import logging
import os
import stat
import subprocess
import tarfile
import tempfile
import urllib2


def YumInstall(package_list):
  if YumInstall.first_run:
    Execute(['yum', 'update'])
    YumInstall.first_run = False
  Execute(['yum', '-y', 'install'] + package_list)
YumInstall.first_run = True


def AptGetInstall(package_list):
  if AptGetInstall.first_run:
    try:
      Execute(['apt-get', 'update'])
    except subprocess.CalledProcessError as error:
      logging.warning('Apt update failed, trying again: %s', error)
      Execute(['apt-get', 'update'], raise_errors=False)
    AptGetInstall.first_run = False

  env = os.environ.copy()
  env['DEBIAN_FRONTEND'] = 'noninteractive'
  return Execute(['apt-get', '-q', '-y', 'install'] + package_list, env=env)
AptGetInstall.first_run = True


def PipInstall(package_list):
  """Install Python modules via pip. Assumes pip is already installed."""
  return Execute(['pip', 'install', '-U'] + package_list)


def GemInstall(gem_list):
  """Installs ruby gems, assumes ruby and rubygems is already installed."""
  return Execute(['gem', 'install'] + gem_list)


def Gsutil(params):
  """Call gsutil."""
  env = os.environ.copy()
  return Execute(['gsutil'] + params, capture_output=True, env=env)


def Execute(cmd, cwd=None, capture_output=False, env=None, raise_errors=True):
  """Execute an external command (wrapper for Python subprocess)."""
  logging.info('Command: %s', str(cmd))
  returncode = 0
  output = None
  try:
    if capture_output:
      output = subprocess.check_output(cmd, cwd=cwd, env=env)
    else:
      subprocess.check_call(cmd, cwd=cwd, env=env)
  except subprocess.CalledProcessError as e:
    if raise_errors:
      raise
    else:
      returncode = e.returncode
      output = e.output
      logging.exception('Command returned error status %d', returncode)
  if output:
    logging.info(output)
  return returncode, output, None


def ParseGitUrlBranchCombo(combo):
  """Parses the URL[;branch] combo we use for Git URLs.

  This does assume the URL doesn't contain a semicolon. Some separator had to be
  chosen to allow for convenient command-line input. Most others are worse.

  Args:
    combo: either a bare URL or an URL:branch

  Returns:
    Always returns two values, the first of which is the URL. The second will be
    a branch if one was given and None otherwise.
  """

  url, _, branch = combo.partition(';')
  # str.partition() will set branch to '' when we want None.
  return url, (branch if branch else None)


def GitClone(url, branch=None, target_dir=None):
  """Clones a git repo and returns the directory name. Git must be present."""

  target_dir = target_dir if target_dir else CreateTempDirectory()
  if branch:
    Execute(['git', 'clone', '-n', url], cwd=target_dir)
    git_basedir = os.path.join(target_dir, os.listdir(target_dir)[0])
    Execute(['git', 'fetch', 'origin', branch], cwd=git_basedir)
    Execute(['git', 'checkout', 'FETCH_HEAD'], cwd=git_basedir)
    logging.info('Checked out branch %s of repo %s into directory %s',
                 branch, url, git_basedir)
  else:
    Execute(['git', 'clone', url], cwd=target_dir)
    git_basedir = os.path.join(target_dir, os.listdir(target_dir)[0])
    logging.info('Checked out %s into directory %s', url, git_basedir)
  return git_basedir


def GitMergeFromUrl(repo_dir, url, branch):
  """Merges a remote git repository by URL and branch into a local checkout."""

  Execute(['git', 'pull', '--commit', '--no-edit', url, branch], cwd=repo_dir)


def DownloadAndExtractGithubTarball(url):
  # Github tarballs are a snapshot of the codebase from HEAD. The archive's root
  # is a single directory with the codebase inside. Append the directory to the
  # base dir to get the path of the code.
  return DownloadAndExtractTarball(url, cd_first_directory=True)


def DownloadAndExtractTarball(url, cd_first_directory=False):
  """Downloads a .tar.gz file and extracts it to a directory."""
  target_dir = CreateTempDirectory()
  tmp_dir = CreateTempDirectory()
  tarball_path = os.path.join(tmp_dir, 'file.tar.gz')
  DownloadFile(url, tarball_path)
  UntarGzFile(tarball_path, target_dir)
  logging.info('Contents of archive %s', url)
  Execute(['ls', target_dir])
  # Getting the first directory is useful for github and upstream tarballs
  # because extract they normally have a single directory as the root.
  if cd_first_directory:
    return os.path.join(target_dir, os.listdir(target_dir)[0])
  else:
    return target_dir


def UntarGzFile(tarball_path, output_dir):
  tf = tarfile.open(tarball_path, 'r:gz')
  try:
    tf.extractall(output_dir)
  finally:
    tf.close()


def InstallSetupPyRequirements(package_type):
  if package_type == 'deb':
    AptGetInstall(['python-stdeb', 'gcc', 'python-dev', 'python-setuptools',
                   'dh-python'])
  elif package_type == 'rpm':
    YumInstall(['rpm-build', 'gcc', 'python-devel', 'python-setuptools'])


def BuildPythonPackage(package_type, src_dir):
  """Generates a DEB package from a setuptools based Python package."""
  if package_type == 'deb':
    Execute(['python', 'setup.py', '--command-packages=stdeb.command',
             'sdist_dsc', '--debian-version', '1', 'bdist_deb'],
            cwd=src_dir)
    return glob.glob(os.path.join(src_dir, 'deb_dist', '*.deb'))[0]
  elif package_type == 'rpm':
    Execute(['python', 'setup.py', 'bdist_rpm'],
            cwd=src_dir)
    # Get the first RPM that's not the source RPM.
    return [item for item in glob.glob(os.path.join(src_dir, 'dist', '*.rpm'))
            if '.src.' not in item][0]
  return None


def CreateTempDirectory():
  return tempfile.mkdtemp(prefix='builder')


def DownloadFile(remote_path, local_path, params=None):
  if not params:
    params = []
  if remote_path.startswith('gs://'):
    Gsutil(['cp', remote_path, local_path])
  else:
    Execute(['curl', '-o', local_path, '-L', remote_path] + params)


def HttpGetJson(url, headers=None):
  return json.loads(HttpGet(url, headers=headers))


def HttpGet(url, headers=None):
  request = urllib2.Request(url)
  if headers:
    for key in headers.keys():
      request.add_unredirected_header(key, headers[key])
  return urllib2.urlopen(request).read()


def GetMetadataParam(name, default_value=None, raise_on_not_found=False):
  try:
    url = 'http://metadata/computeMetadata/v1/instance/attributes/%s' % name
    return HttpGet(url, headers={'Metadata-Flavor': 'Google'})
  except urllib2.HTTPError:
    if raise_on_not_found:
      raise ValueError('Metadata key "%s" not found' % name)
    else:
      return default_value


def GetMetadataParamBool(name, default_value):
  value = GetMetadataParam(name, default_value)
  if not value:
    return False
  return True if value.lower() == 'yes' else False


def MakeExecutable(file_path):
  os.chmod(file_path, os.stat(file_path).st_mode | stat.S_IEXEC)


def ReadFile(file_path, strip=False):
  content = open(file_path).read()
  if strip:
    return content.strip()
  return content


def WriteFile(file_path, content, mode='w'):
  with open(file_path, mode) as fp:
    fp.write(content)


def CreateTarGzFromDirectory(dir_path, tarball_file):
  try:
    tarball = tarfile.open(tarball_file, 'w:gz')
    tarball.add(dir_path, arcname='')
  finally:
    tarball.close()


def SetupLogging():
  logfile = tempfile.mktemp(dir='/tmp', prefix='startupscript_')
  logging_level = logging.DEBUG
  logging.basicConfig(filename=logfile, level=logging_level)
  console = logging.StreamHandler()
  console.setLevel(logging_level)
  logging.getLogger().addHandler(console)


def RunScript(script_func):
  try:
    script_func()
  except BaseException, e:
    logging.exception(e)
    raise

SetupLogging()
