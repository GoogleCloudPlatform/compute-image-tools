# Copyright 2017 Google Compute Engine Guest OS team.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
import sys

# Change the default logging format.
# INFO 20170809 14:57:37.432  137043 file.py:853] foo!
_fmt = '%(levelname)s %(asctime)s.%(msecs)d  %(thread)d %(filename)s:%(lineno)d] %(message)s'
logging._defaultFormatter._fmt = _fmt
logging._defaultFormatter.datefmt = '%Y%I%d %H:%M:%S'

_out_handler = logging.StreamHandler(sys.stdout)
_err_handler = logging.StreamHandler(sys.stderr)
_out_handler.setLevel(logging.INFO)
_err_handler.setLevel(logging.ERROR)
_logger = logging.getLogger(__name__)
_logger.addHandler(_out_handler)
_logger.addHandler(_err_handler)
_logger.setLevel(logging.INFO)
