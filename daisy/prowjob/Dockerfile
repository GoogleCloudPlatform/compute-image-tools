FROM python:2-alpine

ENV GOOGLE_APPLICATION_CREDENTIALS /etc/service-account/service-account.json

RUN pip install -U pip --no-cache-dir
RUN pip install google-auth google-cloud-storage --no-cache-dir

RUN apk update
RUN apk add ca-certificates --no-cache
RUN update-ca-certificates
RUN apk add git openssl --no-cache

## Install Go
RUN wget https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz && \
    tar -C / -xzf go1.9.linux-amd64.tar.gz && \
    rm -rfv go1.9.linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH /go/bin:$PATH

## Installation cleanup
RUN apk del openssl
RUN rm -rf /var/cache/apk/*
RUN pip uninstall -y pip

WORKDIR /workdir
ENV PYTHONPATH /workdir:$PYTHONPATH
COPY run run

ENTRYPOINT ["python", "-m", "run"]
