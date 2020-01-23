# Enterprise Linux Image Builds

Base image builds for GCE RHEL, CentOS, and Oracle Linux 6 and 7.

Red Hat BYOL builds will install subscription-manager in the resulting image to
attach a Red Hat subscription to the resulting instance. The build itself will
not attach a subscription to the image.

Example Daisy invocations:
```shell

# CentOS 7 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=CentOS-7-x86_64-DVD-1611.iso \
      centos_7.wf.json

# RHEL 7
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=rhel-server-7.3-x86_64-dvd.iso \
      rhel_7.wf.json

# RHEL 7 BYOS
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=rhel-server-7.3-x86_64-dvd.iso \
      rhel_7_byos.wf.json

# Oracle Linux 7
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=OracleLinux-R7-U4-Server-x86_64-dvd.iso \
      -var:version_major=7 \
      oraclelinux7.wf.json
```
