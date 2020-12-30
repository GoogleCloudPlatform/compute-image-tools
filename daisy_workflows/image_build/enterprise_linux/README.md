# Enterprise Linux Image Builds

Base image builds for GCE Enterprise Linux based builds (RHEL, CentOS, etc).

Red Hat BYOS builds will install subscription-manager in the resulting image to
attach a Red Hat subscription to the resulting instance. The build itself will
not attach a subscription to the image.

Example Daisy invocations:
```shell

# CentOS 7 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=CentOS-7-x86_64-DVD-2009.iso \
      centos_7.wf.json

# RHEL 7
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=rhel-server-7.9-x86_64-dvd.iso \
      rhel_7.wf.json

# RHEL 7 BYOS
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=rhel-server-7.9-x86_64-dvd.iso \
      rhel_7_byos.wf.json
```
