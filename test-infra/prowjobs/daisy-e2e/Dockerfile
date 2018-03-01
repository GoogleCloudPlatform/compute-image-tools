# Copyright 2017 Google Inc. All Rights Reserved.
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
FROM gcr.io/compute-image-tools-test/wrapper:latest

FROM python:alpine

ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json

RUN apk add --no-cache gcc git libffi-dev musl-dev openssl-dev

# Set up Python packages.
COPY requirements.txt requirements.txt
RUN pip3 install -U pip --no-cache-dir
RUN pip3 install -r requirements.txt --no-cache-dir

# Set up test runner working directory.
WORKDIR /
COPY --from=0 /wrapper wrapper
ENV PYTHONPATH /:$PYTHONPATH
COPY run run
ENTRYPOINT ["./wrapper", "python3", "-m", "run"]
