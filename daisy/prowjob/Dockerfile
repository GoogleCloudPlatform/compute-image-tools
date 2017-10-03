FROM debian:stretch-slim

RUN apt-get update
RUN apt-get install -y git libffi-dev libssl-dev python python-pip wget

ENV CODECOV_TOKEN /etc/codecov/token
ENV GOOGLE_APPLICATION_CREDENTIALS /etc/service-account/service-account.json
ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV GOVERSION 1.9
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH

# Set up Go.
RUN mkdir -p $GOPATH $(dirname $GOROOT)
RUN wget -O go.tar.gz https://storage.googleapis.com/golang/go$GOVERSION.linux-amd64.tar.gz && \
    tar -xzf go.tar.gz -C $(dirname $GOROOT) && \
    rm go.tar.gz

# Set up Python packages.
COPY requirements.txt requirements.txt
RUN pip install -U pip --no-cache-dir
RUN pip install -r requirements.txt --no-cache-dir

# Set up test runner working directory.
WORKDIR /workdir
ENV PYTHONPATH /workdir:$PYTHONPATH
COPY run run

ENTRYPOINT ["python", "-m", "run"]