#!/usr/bin/env python3
# Copyright 2020 Google Inc. All Rights Reserved.
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
"""Perform inspection, and print results to stdout."""

import argparse
import json
import urllib.request

from boot_inspect import inspection
from boot_inspect import model
import guestfs


def _daisy_kv(key: str, value: str):
  template = "Status: <serial-output key:'{key}' value:'{value}'>"
  return template.format(key=key, value=value)


def _output_daisy(results: model.InspectionResults):
  if results:
    if results.architecture:
      print(_daisy_kv('architecture', results.architecture.value))
    if results.os:
      print(_daisy_kv('distro', results.os.distro.value))
      print(_daisy_kv('major', results.os.version.major))
      print(_daisy_kv('minor', results.os.version.minor))
    if results.bios_bootable:
      print(_daisy_kv('bios_bootable', 'true'))
    if results.uefi_bootable:
      print(_daisy_kv('uefi_bootable', 'true'))
    if results.root_fs:
      print(_daisy_kv('root_fs', results.root_fs))
  print('Success: Done!')


def _output_json(results: model.InspectionResults, indent=None):
  print(json.dumps(results, indent=indent,
                   cls=model.ModelJSONEncoder))


def _output_human(results: model.InspectionResults):
  _output_json(results, indent=4)


def main():
  format_options_and_help = {
    'json': 'JSON without newlines. Suitable for consumption by '
            'another program.',
    'human': 'Readable format that includes newlines and indents.',
    'daisy': 'Key-value format supported by Daisy\'s serial log collector.',
  }

  parser = argparse.ArgumentParser(
    description='Find boot-related properties of a disk.')
  parser.add_argument(
    '--format',
    choices=format_options_and_help.keys(),
    default='human',
    help=' '.join([
      '`%s`: %s' % (key, value)
      for key, value in format_options_and_help.items()
    ]))
  parser.add_argument(
    'device',
    help='a block device or disk file.'
  )
  args = parser.parse_args()

  g = guestfs.GuestFS(python_return_dict=True)
  g.add_drive_opts(args.device, readonly=1)
  g.launch()

  req = urllib.request.Request(
      "http://metadata.google.internal/computeMetadata/v1"
      "/instance/attributes/is-inspect-os",
      headers={'Metadata-Flavor': 'Google'})
  is_inspect_os = urllib.request.urlopen(req).read()
  if is_inspect_os == b'true':
    results = inspection.inspect_device(g, args.device)
  else:
    results = model.InspectionResults(device=None, os=None, architecture=None)

  boot_results = inspection.inspect_boot_loader(g)
  results.bios_bootable = boot_results.bios_bootable
  results.uefi_bootable = boot_results.uefi_bootable
  results.root_fs = boot_results.root_fs

  globals()['_output_' + args.format](results)


if __name__ == '__main__':
  main()
