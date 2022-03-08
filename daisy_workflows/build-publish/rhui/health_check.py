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
import sys
import tempfile
import typing


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
      for line in mounts:
        parts = line.split()
        mnt_point, mnt_type = parts[1], parts[2]
        if mnt_point == self.nfs_mount:
          if mnt_type == "nfs":
            return
          else:
            raise EnvironmentError("Not mounted as nfs. mount: %s" % line)
    raise EnvironmentError("%s not in /proc/mounts" % self.nfs_mount)

  def mount_is_writable(self, logger: logging.Logger):
    logger.info("checking that %s is writable", self.nfs_mount)
    with tempfile.TemporaryFile(dir=self.nfs_mount) as f:
      f.write(b"content")

  def mount_is_readable(self, logger: logging.Logger):
    logger.info("checking that %s is readable", self.nfs_mount)
    os.listdir(self.nfs_mount)


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
    nargs=1,
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
  parser.add_argument(
    "--nfs_mount",
    dest="nfs_mount",
    type=str,
    nargs=1,
    required=True,
    help="directory of NFS mount",
  )
  parser.add_argument(
    "--log",
    dest="log",
    type=str,
    nargs=1,
    choices=["stdout", "syslog"],
    default="syslog",
    help="where to write logs",
  )
  args = parser.parse_args()
  if args.log[0] == "stdout":
    handler = logging.StreamHandler(sys.stdout)
  else:
    handler = logging.handlers.SysLogHandler(address="/dev/log")
    # syslog parses on the colon to determine the tag for the message.
    handler.ident = "rhui-health-check: "
  logging.basicConfig(
    level=logging.DEBUG,
    format="%(name)s - %(levelname)s - %(message)s",
    handlers=[handler],
  )
  main(
    node_type=args.node[0],
    result_file=args.result_file,
    nfs_mount=args.nfs_mount[0],
  )
