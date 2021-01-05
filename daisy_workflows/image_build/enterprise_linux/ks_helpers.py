#!/usr/bin/env python3
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

"""Kickstart helper functions used to build kickstart files."""

import logging
import os


class RepoString(object):
  """Creates a yum.conf repository section statement for a kickstart file.

  See the yum.conf man pages for more information about formatting
  requirements.

  The attributes listed are the minimun data set for a repo section.

  Attributes:
    head: The header that should be used for the repo section.
    name: The name as it will appear in yum.
    baseurl: The url for the repo.
    enabled: Set to 1 to enable.
    gpgcheck: Set to 1 to enable.
    repo_gpgcheck: Set to 1 to enable.
    gpgkey: URLs pointing to GPG keys.
  """

  url_root = 'https://packages.cloud.google.com/yum/repos'
  gpgkey_list = [
      'https://packages.cloud.google.com/yum/doc/yum-key.gpg',
      'https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg'
  ]

  # New repos should be added here. Create a dict for your repo bellow.
  # This dict should contain the following:
  # head: The header that should be used for the repo section.
  # name: The name as it will appear in yum.
  # url_branch: This is combined with url_root (defined in the class) and
  #   repo_version to create the repo's baseurl. You must include a string
  #   formatter '%s' to place the repo_version in your URL.
  #   e.g. /google-compute-engine-%s-x86_64-unstable
  # filename: This is the location the yum.conf section file will live on the
  # image.

  repodict = {
      'stable': {
          'head': '[google-compute-engine]',
          'name': 'Google Compute Engine',
          'url_branch': '/google-compute-engine-%s-x86_64-stable',
          'filename': '/etc/yum.repos.d/google-cloud.repo'
      },
      'sdk': {
          'head': '[google-cloud-sdk]',
          'name': 'Google Cloud SDK',
          'url_branch': '/cloud-sdk-%s-x86_64',
          'filename': '/etc/yum.repos.d/google-cloud.repo'
      },
      'unstable': {
          'head': '[google-compute-engine-unstable]',
          'name': 'Google Compute Engine Unstable',
          'url_branch': '/google-compute-engine-%s-x86_64-unstable',
          'filename': '/etc/yum.repos.d/google-cloud-unstable.repo'
      },
      'staging': {
          'head': '[google-compute-engine-staging]',
          'name': 'Google Compute Engine Staging',
          'url_branch': '/google-compute-engine-%s-x86_64-staging',
          'filename': '/etc/yum.repos.d/google-cloud-staging.repo'
      }
  }

  def __init__(self, repo_version, repo):
    """Initializes RepoString with attributes passes as arguments.

    Args:
      repo_version: string; expects 'el7', 'el8'.

      repo: string; used to specify which dict in repodict to use to assemble
            the yum.conf repo segment.

    repodata must contain the following:
    head: The header that should be used for the repo entry.
    name: The name as it will appear in yum.
    url_branch: This is combined with url_root (defined in the class) and
    repo_version to create the repo's baseurl. You must include a string
      formatter '%s' to place the repo_version in your URL.
      e.g. /google-compute-engine-%s-x86_64-unstable

    Returns:
      An initialized RepoString object.
    """
    super(RepoString, self).__init__()
    self.repo = repo
    self.repo_version = repo_version
    self.yumseg = {}
    self.yumseg['head'] = self.repodict[self.repo]['head']
    self.yumseg['name'] = self.repodict[self.repo]['name']
    self.yumseg['baseurl'] = (
        self.GetBaseURL(self.repodict[self.repo]['url_branch']))
    self.yumseg['enabled'] = '1'
    self.yumseg['gpgcheck'] = '1'
    self.yumseg['repo_gpgcheck'] = '1'
    self.yumseg['gpgkey'] = self.gpgkey_list

  def __str__(self):
    """Override the string method to return a yum.conf repository section.

    Returns:
      RepoString; tell python to treat this as a string using str().
    """
    keylist = ['head',
               'name',
               'baseurl',
               'enabled',
               'gpgcheck',
               'repo_gpgcheck',
               'gpgkey']
    yum_repo_list = (
        [('tee -a %s << EOM' % self.repodict[self.repo]['filename']), ])
    for key in keylist:
      if key == 'head':
        yum_repo_list.append(self.yumseg[key])
      elif key == 'gpgkey':
        yum_repo_list.append('%s=%s' %
                             (key, '\n       '.join(self.gpgkey_list)))
      else:
        yum_repo_list.append('%s=%s' % (key, self.yumseg[key]))
    yum_repo_list.append('EOM')
    return '\n'.join(yum_repo_list)

  def GetBaseURL(self, url_branch):
    """Assembles the baseurl attribute of RepoString.

    Proccesses the string formatting in url_branch then combines it with
    url_root to create the baseurl.

    Args:
      url_branch: string; this is combined with url_root and repo_version to
                  create the repo's baseurl. You must include a string
                  formatter '%s' to place the repo_version in your URL.
                    e.g. /google-compute-engine-%s-x86_64-unstable

    Returns:
      string; baseurl
    """
    return self.url_root + (url_branch % self.repo_version)


def BuildKsConfig(release, google_cloud_repo, byos, sap):
  """Builds kickstart config from shards.

  Args:
    release: string; image from metadata.
    google_cloud_repo: string; expects 'stable', 'unstable', or 'staging'.
    byos: bool; true if using a BYOS RHEL license.
    sap: bool; true if building RHEL for SAP.

  Returns:
    string; a valid kickstart config.
  """

  # Each kickstart file will have the following components:
  # ks_options: kickstart install options
  # ks_packages: kickstart package config
  # ks_post: kickstart post install scripts

  # Common kickstart sections.
  ks_cleanup = FetchConfigPart('cleanup.cfg')
  el7_packages = FetchConfigPart('el7-packages.cfg')
  el8_packages = FetchConfigPart('el8-packages.cfg')
  el7_post = FetchConfigPart('el7-post.cfg')
  el8_post = FetchConfigPart('el8-post.cfg')

  # For BYOS RHEL, don't remove subscription-manager.
  if byos:
    logging.info('Building RHEL BYOS image.')
    rhel_byos_post = FetchConfigPart('rhel-byos-post.cfg')

  # RHEL 7 variants.
  if release.startswith('rhel-7'):
    logging.info('Building RHEL 7 image.')
    repo_version = 'el7'
    ks_options = FetchConfigPart('rhel-7-options.cfg')
    ks_packages = el7_packages
    # Build RHEL post scripts.
    rhel_post = FetchConfigPart('rhel-7-post.cfg')
    if sap:
      logging.info('Building RHEL 7 for SAP')
      rhel_sap_post = FetchConfigPart('rhel-7-sap-post.cfg')
      point = ''
      if release == 'rhel-7-4':
        logging.info('Building RHEL 7.4 for SAP')
        point = FetchConfigPart('rhel-7-4-post.cfg')
      if release == 'rhel-7-6':
        logging.info('Building RHEL 7.6 for SAP')
        point = FetchConfigPart('rhel-7-6-post.cfg')
      if release == 'rhel-7-7':
        logging.info('Building RHEL 7.7 for SAP')
        point = FetchConfigPart('rhel-7-7-post.cfg')
      rhel_post = '\n'.join([point, rhel_sap_post])
    if byos:
      rhel_post = '\n'.join([rhel_post, rhel_byos_post])
    # Combine RHEL specific post install to EL7 post.
    ks_post = '\n'.join([rhel_post, el7_post])

  # RHEL 8 variants.
  elif release.startswith('rhel-8'):
    logging.info('Building RHEL 8 image.')
    repo_version = 'el8'
    ks_options = FetchConfigPart('rhel-8-options.cfg')
    ks_packages = el8_packages
    # Build RHEL post scripts.
    rhel_post = FetchConfigPart('rhel-8-post.cfg')
    if sap:
      logging.info('Building RHEL 8 for SAP')
      rhel_sap_post = FetchConfigPart('rhel-8-sap-post.cfg')
      point = ''
      if release == 'rhel-8-1':
        logging.info('Building RHEL 8.1 for SAP')
        point = FetchConfigPart('rhel-8-1-post.cfg')
      elif release == 'rhel-8-2':
        logging.info('Building RHEL 8.2 for SAP')
        point = FetchConfigPart('rhel-8-2-post.cfg')
      rhel_post = '\n'.join([point, rhel_sap_post])
    if byos:
      rhel_post = '\n'.join([rhel_post, rhel_byos_post])
    # Combine RHEL specific post install to EL8 post.
    ks_post = '\n'.join([rhel_post, el8_post])

 # CentOS 7
  elif release.startswith('centos-7'):
    logging.info('Building CentOS 7 image.')
    repo_version = 'el7'
    ks_options = FetchConfigPart('centos-7-options.cfg')
    ks_packages = el7_packages
    ks_post = el7_post

  # CentOS 8
  elif release.startswith('centos-8'):
    logging.info('Building CentOS 8 image.')
    repo_version = 'el8'
    ks_options = FetchConfigPart('centos-8-options.cfg')
    ks_packages = el8_packages
    ks_post = el8_post

  # CentOS Stream 8
  elif release.startswith('centos-stream-8'):
    logging.info('Building CentOS Stream image.')
    repo_version = 'el8'
    ks_options = FetchConfigPart('centos-stream-8-options.cfg')
    ks_packages = el8_packages
    ks_post = el8_post

  else:
    logging.error('Unknown Image Name: %s' % release)

  # Post section for Google cloud repos.
  repo_post = BuildReposPost(repo_version, google_cloud_repo)
  # Joined kickstart post sections.
  ks_post = '\n'.join([repo_post, ks_post, ks_cleanup])
  # Joine kickstart file.
  ks_file = '\n'.join([ks_options, ks_packages, ks_post])

  # Return the joined kickstart file as a string.
  logging.info("Kickstart file: \n%s", ks_file)
  return ks_file


def BuildReposPost(repo_version, google_cloud_repo):
  """Creates a kickstart post macro with repos needed by GCE.

  Args:
    repo_version: string; expects 'el7', or 'el8'.
    google_cloud_repo: string; expects 'stable', 'unstable', or 'staging'

  Returns:
    string; a complete %post macro that can be added to a kickstart file. The
    output should look like the following example.

    %post
    tee -a /etc/yum.repos.d/example.repo << EOF
    [example-repo]
    name=Example Repo
    baseurl=https://example.com/yum/repos/example-repo-ver-x86_64
    enabled=1
    gpgcheck=1
    repo_gpgcheck=1
    gpgkey=https://example.com/yum/doc/yum-key.gpg
           https://example.com/yum/doc/rpm-package-key.gpg
    EOF
    ...
    %end

  The values for enabled, gpgcheck, repo_gpgcheck, and gpgkey are constants.
  the values for head, name, and baseurl can be modified to point to use any
  repo that will accept the supplied gpg keys.
  """

  # Build a list of repos that will be returned. All images will get the
  # compute repo. EL7 images get the cloud SDK repo. The unstable, and staging
  # repos can be added to either by setting the google_cloud_repo value.
  repolist = ['stable']
  if repo_version == 'el7' or repo_version == 'el8':
    repolist.append('sdk')
  if google_cloud_repo == 'unstable':
    repolist.append('unstable')
  if google_cloud_repo == 'staging':
    repolist.append('staging')

  filelist = ['%post']
  for repo in repolist:
    filelist.append(str(RepoString(repo_version, repo)))
  filelist.append('%end')
  return '\n'.join(filelist)


def FetchConfigPart(config_file):
  """Reads data from a kickstart file.

  Args:
    config_file: string; the name of a kickstart file shard located in
        the 'kickstart' directory.

  Returns:
    string; contents of config_file should be a string with newlines.
  """
  with open(os.path.join('files', 'kickstart', config_file)) as f:
    return f.read()
