import tasks
from bootstrapvz.common.tools import rel_path


def validate_manifest(data, validator, error):
  validator(data, rel_path(__file__, 'manifest-schema.yml'))


def resolve_tasks(taskset, manifest):
  if manifest.plugins['google_cloud_test_repos'].get('staging', False):
    taskset.add(tasks.AddGoogleCloudStagingRepo)
  if manifest.plugins['google_cloud_test_repos'].get('unstable', False):
    taskset.add(tasks.AddGoogleCloudUnstableRepo)
  taskset.add(tasks.CleanupGoogleCloudTestRepos)
