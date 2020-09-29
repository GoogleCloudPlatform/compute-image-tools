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
import os
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


def _inspect_boot_loader(device):
  bios_bootable = False
  uefi_bootable = False
  root_fs = ""

  try:
    g = guestfs.GuestFS(python_return_dict=True)
    g.add_drive_opts(device, readonly=1)
    g.launch()

    part_list = g.part_list('/dev/sda')
    for part in part_list:
      guid = g.part_get_gpt_type('/dev/sda', part['part_num'])

      # It covers both GPT "EFI System" and BIOS "EFI (FAT-12/16/32)"
      if guid == 'C12A7328-F81F-11D2-BA4B-00A0C93EC93B':
        uefi_bootable = True
        # TODO: detect root_fs (b/169245755)
      if guid == '21686148-6449-6E6F-744E-656564454649':
        bios_bootable = True

  except Exception as e:
    print("Failed to inspect disk partition: ", e)

  return bios_bootable, uefi_bootable, root_fs


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

  req = urllib.request.Request(
      "http://metadata.google.internal/computeMetadata/v1"
      "/instance/attributes/is-inspect-os",
      headers={'Metadata-Flavor': 'Google'})
  is_inspect_os = urllib.request.urlopen(req).read()
  if is_inspect_os == b'true':
    g = guestfs.GuestFS(python_return_dict=True)
    g.add_drive_opts(args.device, readonly=1)
    g.launch()
    results = inspection.inspect_device(g, args.device)
  else:
    results = model.InspectionResults(device=None, os=None, architecture=None)

  results.bios_bootable, results.uefi_bootable, results.root_fs = (
      _inspect_boot_loader(args.device))

  globals()['_output_' + args.format](results)


if __name__ == '__main__':
  main()
