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
FROM gcr.io/$PROJECT_ID/wrapper:latest

FROM golang

ENV ARTIFACTS /artifacts
RUN mkdir -p ${ARTIFACTS}
ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json
ENV CODECOV_TOKEN /etc/codecov/codecov

RUN apt-get update && apt-get -y install libssl-dev && \
    rm -rf /var/cache/apt/archives

# Go test junit xml output
RUN go install github.com/jstemmer/go-junit-report@latest
ENV GO111MODULE on

COPY --from=0 /wrapper wrapper
COPY main.sh main.sh
ENTRYPOINT ["./wrapper", "./main.sh"]
