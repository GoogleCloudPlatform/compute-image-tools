"""Adds Google Cloud test repos."""

import os

from bootstrapvz.base import Task
from bootstrapvz.common import phases
from bootstrapvz.common.tasks import apt
from bootstrapvz.common.tools import log_check_call


class AddGoogleCloudStagingRepo(Task):
  description = 'Adding Google Cloud Staging Repo'
  phase = phases.preparation
  predecessors = [apt.AddManifestSources]

  @classmethod
  def run(cls, info):
    info.source_lists.add(
        'google-cloud-test',
        'deb http://packages.cloud.google.com/apt '
        'google-compute-engine-{system.release}-staging main')


class AddGoogleCloudUnstableRepo(Task):
  description = 'Adding Google Cloud Unstable Repo'
  phase = phases.preparation
  predecessors = [apt.AddManifestSources]

  @classmethod
  def run(cls, info):
    info.source_lists.add(
        'google-cloud-test',
        'deb http://packages.cloud.google.com/apt '
        'google-compute-engine-{system.release}-unstable main')


class CleanupGoogleCloudTestRepos(Task):
  description = 'Removing Google Cloud Test Repos'
  phase = phases.system_cleaning
  successors = [apt.AptClean]

  @classmethod
  def run(cls, info):
    staging_list = os.path.join(
        info.root, 'etc/apt/sources.list.d',
        'google-cloud-test.list')
    log_check_call(['rm', '-f', staging_list])
