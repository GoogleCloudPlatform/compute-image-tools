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

import inspection


def _daisy_kv(key: str, value: str):
  template = "Status: <serial-output key:'{key}' value:'{value}'>"
  return template.format(key=key, value=value)


def _output_daisy(results: inspection.model.InspectionResults):
  if results:
    print(_daisy_kv('architecture', results.architecture.value))
    print(_daisy_kv('distro', results.os.distro.value))
    print(_daisy_kv('major', results.os.version.major))
    print(_daisy_kv('minor', results.os.version.minor))
  print('Success: Done!')


def _output_json(results: inspection.model.InspectionResults, indent=None):
  print(json.dumps(results, indent=indent,
                   cls=inspection.model.ModelJSONEncoder))


def _output_human(results: inspection.model.InspectionResults):
  _output_json(results, indent=4)


def main(device: str, fmt: str):
  results = inspection.inspect_device(device)
  globals()['_output_' + fmt](results)


if __name__ == '__main__':
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
  main(device=args.device, fmt=args.format)
