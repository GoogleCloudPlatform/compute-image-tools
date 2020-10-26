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
import base64

from boot_inspect import inspection
from compute_image_tools_proto import inspect_pb2
from google.protobuf import text_format
from google.protobuf.json_format import MessageToJson
import guestfs


def _daisy_kv(key: str, value: str):
  template = "Status: <serial-output key:'{key}' value:'{value}'>"
  return template.format(key=key, value=value)


def _output_daisy(results: inspect_pb2.InspectionResults):
  if results:
    print('Results: ')
    print(text_format.MessageToString(results))
    print(
        _daisy_kv(
            'inspect_pb',
            base64.standard_b64encode(results.SerializeToString()).decode()))
  print('Success: Done!')


def _output_human(results: inspect_pb2.InspectionResults):
  print(MessageToJson(results, indent=4))


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
  parser.add_argument(
    '--inspect-os',
    help='whether to detect the operating system.',
    action='store_true'
  )
  args = parser.parse_args()
  results = inspect_pb2.InspectionResults()
  try:
    g = guestfs.GuestFS(python_return_dict=True)
    g.add_drive_opts(args.device, readonly=1)
    g.launch()
  except BaseException as e:
    print('Failed to mount guest: ', e)
    results.ErrorWhen = inspect_pb2.InspectionResults.ErrorWhen.MOUNTING_GUEST
    return results

  if args.inspect_os:
    try:
      print('Inspecting OS')
      results = inspection.inspect_device(g)
    except BaseException as e:
      print('Failed to inspect OS: ', e)
      results.ErrorWhen = inspect_pb2.InspectionResults.ErrorWhen.INSPECTING_OS
      return results

  try:
    boot_results = inspection.inspect_boot_loader(g, args.device)
  except BaseException as e:
    print('Failed to inspect boot loader: ', e)
    results.ErrorWhen = \
        inspect_pb2.InspectionResults.ErrorWhen.INSPECTING_BOOTLOADER
    return results

  results.bios_bootable = boot_results.bios_bootable
  results.uefi_bootable = boot_results.uefi_bootable
  results.root_fs = boot_results.root_fs
  globals()['_output_' + args.format](results)


if __name__ == '__main__':
  main()
