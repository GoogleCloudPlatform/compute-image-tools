FROM python:3.6.4-alpine3.7

ENV GOOGLE_APPLICATION_CREDENTIALS /etc/compute-image-tools-test-service-account/creds.json

RUN apk add --no-cache gcc git libffi-dev musl-dev openssl-dev

# Set up Python packages.
COPY requirements.txt requirements.txt
RUN pip3 install -U pip --no-cache-dir
RUN pip3 install -r requirements.txt --no-cache-dir

# Set up test runner working directory.
WORKDIR /workdir
ENV PYTHONPATH /workdir:$PYTHONPATH
COPY run run

ENTRYPOINT ["python3", "-m", "run"]
