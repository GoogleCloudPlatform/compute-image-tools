#  Copyright 2022 Google Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http:#www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
"""Health check for RHUI infrastructure nodes."""

import argparse
import logging
import logging.handlers
import os
import subprocess
import sys
import tempfile
import typing

NFS_MOUNT = "/var/lib/rhui/remote_share"


class HealthChecks:
  nfs_mount: str

  def __init__(
      self,
      nfs_mount: str,
  ) -> None:
    self.nfs_mount = nfs_mount

  def mount_uses_nfs(self, logger: logging.Logger):
    logger.info("checking that %s is mounted as NFS", self.nfs_mount)
    with open("/proc/mounts") as mounts:
        if "/var/lib/rhui/remote_share nfs rw" not in mounts:
            raise EnvironmentError("%s not mounted")

  def mount_is_writable(self, logger: logging.Logger):
    logger.info("checking that %s is writable", self.nfs_mount)
    with tempfile.TemporaryFile(dir=self.nfs_mount) as f:
      f.write(b"content")

  def mount_is_readable(self, logger: logging.Logger):
    logger.info("checking that %s is readable", self.nfs_mount)
    os.listdir(self.nfs_mount)

  def cds_services_are_active(self, logger: logging.Logger):
    logger.info("checking that services are active")
    self._check_services(["gunicorn-auth",
                          "gunicorn-mirror",
                          "gunicorn-content_manager",
                          "nginx"])

  def rhua_services_are_active(self, logger: logging.Logger):
    logger.info("checking that services are active")
    self._check_services(["postgresql",
                          "redis",
                          "pulpcore-api",
                          "pulpcore-content",
                          "pulpcore-resource-manager",
                          "pulpcore-worker@1",
                          "pulpcore-worker@2",
                          "pulpcore-worker@3",
                          "pulpcore-worker@4",
                          "pulpcore-worker@5",
                          "pulpcore-worker@6",
                          "pulpcore-worker@7",
                          "pulpcore-worker@8",
                          "nginx"])

  def _check_services(self, services):
    for service in services:
      subprocess.run(["systemctl", "is-active", service],
                     stdout=subprocess.PIPE,
                     stderr=subprocess.PIPE,
                     check=True)


def main(node_type: str, result_file: typing.TextIO, nfs_mount: str):
  """Executes health check for node_type, and writes the overall
  result to result_file ('OK' if all checks pass, otherwise 'ERROR').
  """
  result_file.truncate()
  health_checks = HealthChecks(nfs_mount=nfs_mount)
  checks = [
    health_checks.mount_is_readable,
    health_checks.mount_uses_nfs,
  ]
  if node_type == "rhua":
    checks += [
      health_checks.mount_is_writable,
      health_checks.rhua_services_are_active,
    ]
  if node_type == "cds":
    checks += [
      health_checks.cds_services_are_active,
    ]
  success = True
  for func in checks:
    logger = logging.getLogger(func.__name__)
    try:
      func(logger=logger)
      logger.info("success")
    except Exception as e:
      success = False
      logger.error(e)
  if success:
    result_file.write("HEALTHY\n")
  else:
    result_file.write("ERROR\n")


if __name__ == "__main__":
  parser = argparse.ArgumentParser(description="Run health checks.")
  parser.add_argument(
    "--node",
    dest="node",
    type=str,
    required=True,
    choices=["cds", "rhua"],
    help="type of node where check is running",
  )
  parser.add_argument(
    "--result_file",
    dest="result_file",
    required=True,
    type=argparse.FileType("w"),
    help="file to write result",
  )
  args = parser.parse_args()

  handler = logging.StreamHandler(sys.stdout)
  logging.basicConfig(
    level=logging.DEBUG,
    format="%(name)s - %(levelname)s - %(message)s",
    handlers=[handler],
  )

  main(args.node, args.result_file, NFS_MOUNT)
