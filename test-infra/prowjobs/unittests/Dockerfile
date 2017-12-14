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
FROM debian:stretch-slim

ENV CODECOV_TOKEN /etc/codecov/token
ENV GOVERSION 1.9.2
ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH ${GOROOT}/bin:${GOPATH}/bin:${PATH}

RUN apt-get update && apt-get -y install bash ca-certificates curl git libssl-dev wget && \
    rm -rf /var/cache/apt/archives

# Install Go.
RUN wget http://redirector.gvt1.com/edgedl/go/go${GOVERSION}.linux-amd64.tar.gz -O go.tgz && \
    mkdir ${GOROOT} ${GOPATH} && \
    tar xf go.tgz && \
    mv go/* ${GOROOT}/ && \
    rm -rf go.tgz go

WORKDIR /
COPY main.sh /main.sh
ENTRYPOINT /main.sh
