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
  #   e.g. /google-cloud-compute-%s-x86_64-unstable
  # filename: This is the location the yum.conf section file will live on the
  # image.

  repodict = {
      'compute': {
          'head': '[google-cloud-compute]',
          'name': 'Google Cloud Compute',
          'url_branch': '/google-cloud-compute-%s-x86_64',
          'filename': '/etc/yum.repos.d/google-cloud.repo'
      },
      'sdk': {
          'head': '[google-cloud-sdk]',
          'name': 'Google Cloud SDK',
          'url_branch': '/cloud-sdk-%s-x86_64',
          'filename': '/etc/yum.repos.d/google-cloud.repo'
      },
      'unstable': {
          'head': '[google-cloud-compute-unstable]',
          'name': 'Google Cloud Compute Unstable',
          'url_branch': '/google-cloud-compute-%s-x86_64-unstable',
          'filename': '/etc/yum.repos.d/google-cloud-unstable.repo'
      },
      'staging': {
          'head': '[google-cloud-compute-staging]',
          'name': 'Google Cloud Compute Staging',
          'url_branch': '/google-cloud-compute-%s-x86_64-staging',
          'filename': '/etc/yum.repos.d/google-cloud-staging.repo'
      }
  }

  def __init__(self, repo_version, repo):
    """Initializes RepoString with attributes passes as arguments.

    Args:
      repo_version: string; expects 'el6', or 'el7'.

      repo: string; used to specify which dict in repodict to use to assemble
            the yum.conf repo segment.

    repodata must contain the following:
    head: The header that should be used for the repo entry.
    name: The name as it will appear in yum.
    url_branch: This is combined with url_root (defined in the class) and
    repo_version to create the repo's baseurl. You must include a string
      formatter '%s' to place the repo_version in your URL.
      e.g. /google-cloud-compute-%s-x86_64-unstable

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
                    e.g. /google-cloud-compute-%s-x86_64-unstable

    Returns:
      string; baseurl
    """
    return self.url_root + (url_branch % self.repo_version)


def BuildKsConfig(release, google_cloud_repo, byol, sap, sap_hana, sap_apps,
                  uefi):
  """Builds kickstart config from shards.

  Args:
    release: string; image from metadata.
    google_cloud_repo: string; expects 'stable', 'unstable', or 'staging'.
    byol: bool; true if using a BYOL RHEL license.
    sap: bool; true if building RHEL for SAP.
    sap_hana: bool; true if building RHEL for SAP HANA.
    sap_apps: bool; true if building RHEL for SAP Apps.
    uefi: bool; true if building uefi image.

  Returns:
    string; a valid kickstart config.
  """

  # This is where we put the kickstart config together. There are three
  # sections in a kickstart config. Sections are:
  # Commands and options
  # Packages
  # pre and post
  # Each section must be in a specific order, but items in that section do not
  # have to be.

  # Common
  pre = ''
  ks_packages = FetchConfigPart('common-packages.cfg')
  # For BYOL RHEL, don't remove subscription-manager.
  if byol:
    logging.info('Building RHEL BYOL image.')
    rhel_byol_post = FetchConfigPart('rhel-byol-post.cfg')

  if release == 'rhel6':
    logging.info('Building RHEL 6 image.')
    pre = FetchConfigPart('el6-pre.cfg')
    ks_options = FetchConfigPart('el6-options.cfg')
    rhel_post = FetchConfigPart('rhel6-post.cfg')
    el_post = FetchConfigPart('el6-post.cfg')
    custom_post = '\n'.join([rhel_post, el_post])
    if byol:
      custom_post = '\n'.join([custom_post, rhel_byol_post])
    cleanup = FetchConfigPart('el6-cleanup.cfg')
    repo_version = 'el6'
  elif release == "centos6":
    logging.info('Building CentOS 6 image.')
    pre = FetchConfigPart('el6-pre.cfg')
    ks_options = FetchConfigPart('el6-options.cfg')
    custom_post = FetchConfigPart('el6-post.cfg')
    cleanup = FetchConfigPart('el6-cleanup.cfg')
    repo_version = 'el6'
  elif release.startswith('rhel7'):
    logging.info('Building RHEL 7 image.')
    if uefi:
      logging.info('Building RHEL 7 for UEFI')
      ks_options = FetchConfigPart('el7-uefi-options.cfg')
    else:
      ks_options = FetchConfigPart('el7-options.cfg')
    rhel_post = FetchConfigPart('rhel7-post.cfg')
    if sap:
      logging.info('Building RHEL 7 for SAP')
      point = ''
      if release == 'rhel7.3':
        logging.info('Building RHEL 7.3 for SAP')
        point = FetchConfigPart('rhel7-3-post.cfg')
      if release == 'rhel7.4':
        logging.info('Building RHEL 7.4 for SAP')
        point = FetchConfigPart('rhel7-4-post.cfg')
      if release == 'rhel7.6':
        logging.info('Building RHEL 7.6 for SAP')
        point = FetchConfigPart('rhel7-6-post.cfg')
      rhel_post = '\n'.join([point, FetchConfigPart('rhel7-sap-post.cfg')])
    elif sap_hana:
      logging.info('Building RHEL 7 for SAP Hana')
      rhel_post = FetchConfigPart('rhel7-sap-hana-post.cfg')
    elif sap_apps:
      logging.info('Building RHEL 7 for SAP Apps')
      rhel_post = FetchConfigPart('rhel7-sap-apps-post.cfg')
    el_post = FetchConfigPart('el7-post.cfg')
    custom_post = '\n'.join([rhel_post, el_post])
    if byol:
      custom_post = '\n'.join([custom_post, rhel_byol_post])
    if uefi:
      rhel_uefi_post = FetchConfigPart('rhel7-uefi-post.cfg')
      custom_post = '\n'.join([custom_post, rhel_uefi_post])
    cleanup = FetchConfigPart('el7-cleanup.cfg')
    repo_version = 'el7'
  elif release == "centos7":
    logging.info('Building CentOS 7 image.')
    ks_options = FetchConfigPart('el7-options.cfg')
    custom_post = FetchConfigPart('el7-post.cfg')
    cleanup = FetchConfigPart('el7-cleanup.cfg')
    repo_version = 'el7'
  elif release == "oraclelinux6":
    logging.info('Building Oracle Linux 6 image.')
    pre = FetchConfigPart('el6-pre.cfg')
    ks_options = FetchConfigPart('el6-options.cfg')
    ol_post = FetchConfigPart('ol6-post.cfg')
    el_post = FetchConfigPart('el6-post.cfg')
    custom_post = '\n'.join([ol_post, el_post])
    cleanup = FetchConfigPart('el6-cleanup.cfg')
    repo_version = 'el6'
  elif release == "oraclelinux7":
    logging.info('Building Oracle Linux 7 image.')
    ks_options = FetchConfigPart('el7-options.cfg')
    ol_post = FetchConfigPart('ol7-post.cfg')
    el_post = FetchConfigPart('el7-post.cfg')
    custom_post = '\n'.join([ol_post, el_post])
    cleanup = FetchConfigPart('el7-cleanup.cfg')
    repo_version = 'el7'
  else:
    logging.error('Unknown Image Name: %s' % release)

  ks_post = BuildPost(custom_post, cleanup, repo_version, google_cloud_repo)

  # This list should be in the order that you want each section to appear in
  # the Kickstart config.
  return '\n'.join([ks_options, ks_packages, pre, ks_post])


def BuildPost(custom_post, cleanup, repo_version, google_cloud_repo):
  """Assembles the %post section of a kickstart file.

  Args:
    custom_post: string; a kickstart %post segment containing post install
                 steps needed for a given flavor of Enterprise Linux.

    cleanup: string; a kickstart %post segment for cleanup.

    repo_version: string; expects 'el6', or 'el7'.

    google_cloud_repo: string; expects 'stable', 'unstable', or 'staging'.

  Returns:
    string; a complete %pre/post segment of a kickstart file.
  """
  # This is used to create a synopsis of the image and should appear right
  # before the image is pushed to GCS.
  create_synopsis = FetchConfigPart('create-synopsis-post.cfg')

  # Configure repository %post section
  repo_post = BuildReposPost(repo_version, google_cloud_repo)

  ks_post_list = [repo_post, custom_post, create_synopsis, cleanup]
  return '\n'.join(ks_post_list)


def BuildReposPost(repo_version, google_cloud_repo):
  """Creates a kickstart post macro with repos needed by GCE.

  Args:
    repo_version: string; expects 'el6', or 'el7'.

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
  repolist = ['compute']
  if repo_version == 'el7':
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
  with open(os.path.join('kickstart', config_file)) as f:
    return f.read()
