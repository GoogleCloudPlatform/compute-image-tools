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

NFS_MOUNT = "/var/lib/rhui/remote_share"


class HealthCheck:

  # Node types the health check supports; defaults to both types.
  nodes = ["cds", "rhua"]

  def __init__(self):
    self.logger = logging.getLogger(self.__class__.__name__)

  def run(self, node: str):
    if node in self.nodes:
      self._run(node)


class MountIsNFS(HealthCheck):

  mount = NFS_MOUNT

  def _run(self, node: str):
    self.logger.info("checking that %s is mounted as NFS", self.mount)
    with open("/proc/mounts") as mounts:
      if f"{self.mount} nfs rw" not in mounts.read():
        raise EnvironmentError("%s is not NFS or not mounted" % self.mount)


class MountIsWritable(HealthCheck):

  mount = NFS_MOUNT

  def _run(self, node: str):
    self.logger.info("checking that %s is writable", self.mount)
    with tempfile.TemporaryFile(dir=self.mount) as f:
      f.write(b"content")


class MountIsReadable(HealthCheck):

  mount = NFS_MOUNT

  def _run(self, node: str):
    self.logger.info("checking that %s is readable", self.mount)
    os.listdir(self.mount)


class ServicesAreActive(HealthCheck):

  cds_services = ["gunicorn-auth",
                  "gunicorn-mirror",
                  "gunicorn-content_manager",
                  "nginx"]

  rhua_services = ["postgresql",
                   "redis",
                   "pulpcore-api",
                   "pulpcore-content",
                   "pulpcore-worker@1",
                   "pulpcore-worker@2",
                   "pulpcore-worker@3",
                   "pulpcore-worker@4",
                   "pulpcore-worker@5",
                   "pulpcore-worker@6",
                   "pulpcore-worker@7",
                   "pulpcore-worker@8",
                   "nginx"]

  def _run(self, node: str):
    self.logger.info("checking that %s services are active", node)
    if node == "cds":
      services = self.cds_services
    if node == "rhua":
      services = self.rhua_services
    for service in services:
      subprocess.run(["systemctl", "is-active", service],
                     stdout=subprocess.PIPE,
                     stderr=subprocess.PIPE,
                     check=True)


class HealthCheckSuite:

  checks = MountIsNFS, MountIsWritable, MountIsReadable, ServicesAreActive

  def run(self, node: str):
    for check_type in self.checks:
      check = check_type()
      try:
        check.run(node)
        check.logger.info("success")
      except Exception as e:
        check.logger.error(e)
        return False

    return True


def main():
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

  args.result_file.truncate()
  if HealthCheckSuite().run(args.node):
    args.result_file.write("HEALTHY\n")
  else:
    args.result_file.write("ERROR\n")


if __name__ == "__main__":
  main()
