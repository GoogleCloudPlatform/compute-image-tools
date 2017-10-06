# Enterprise Linux Image Builds

Base image builds for GCE RHEL and CentOS 6 and 7.

Red Hat BYOL builds will install subscription-manager in the resulting image to
attach a Red Hat subscription to the resulting instance. The build itself will
not attach a subscription to the image.

Note for RHEL 7 builds, you need to provide the point release that matches the
ISO. If these strings do not match, the installer disk will not be bootable.

For example:
If you have the ISO: rhel-server-7.3-x86_64-dvd.iso
The point release is: 7.3


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
      -var:rhel_point_release=7.3 \
      rhel_7.wf.json

# RHEL 7 BYOL
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -var:installer_iso=rhel-server-7.3-x86_64-dvd.iso \
      -var:rhel_point_release=7.3 \
      rhel_7_byol.wf.json
```
