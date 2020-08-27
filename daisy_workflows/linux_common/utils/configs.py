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

import shlex


def update_grub_conf(original: str, **configs: str) -> str:
  """Creates a new grub conf including the specified overrides.

  If a configuration key is already specified in original, then the
  previous configuration is commented.

  Args:
    original: The current grub conf file.
    configs: Key/value pairs to add to the file.
  Returns:
    The new grub conf, encoded as a string.

  """
  new_grub = []
  for line in original.splitlines():
    if '=' in line:
      config, value = line.split('=', 1)
      if config in configs:
        new_grub += [
            '# Removed to support booting on Google Compute Engine.',
            '# ' + line
        ]
        continue
    new_grub.append(line)
  new_grub.append('# Added to support booting on Google Compute Engine.')
  for config, value in configs.items():
    new_grub.append('{}={}'.format(config, shlex.quote(value)))
  return '\n'.join(new_grub)
