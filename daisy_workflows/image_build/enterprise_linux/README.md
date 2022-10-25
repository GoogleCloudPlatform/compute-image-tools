# Enterprise Linux Image Builds

This directory contains [Daisy] workflows and kickstart configs to build the
Enterprise Linux (RHEL, CentOS, etc.) [Public Images] for GCE.

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Details

Enterprise Linux workflows require an installation media to be provided in ISO
format and stored in GCS. The build method is described below, and the output is
a GCE Image in the build project.

### Build method

The specific image build workflows all include the base
`enterprise_linux.wf.json` workflow. This workflow takes the following steps:

1. Create an instance using the `debian-10-worker` image for the boot disk, an
   empty second disk attached, and the startup script set in metadata to use the
   `linux\_common` bootstrap script and the `build\_installer.py` script on boot.

1. On boot, the `build\_installer.py` script will partition and format the empty
   disk, then extract the installer ISO media and the appropriate kickstart
   config onto it. This makes the installer into a bootable disk.

1. Create an instance using the just prepared disk as a boot disk and an empty
   second disk.

1. On boot, the installer will automatically install the operating system onto
   the second disk.

1. Produce an image from the second disk.

### RHEL images

RHEL images will contain by default access to an appropriate set of RHEL
content; use of this content is provided as part of your blah contract and
[more details](here).

Red Hat BYOS images will contain the `subscription-manager` tool which can be
used after boot to attach a Red Hat subscription to the resulting instance. The
build itself will not attach a subscription or include any content access to the
image.

## Invoking the build workflows

Example Daisy invocations:
```shell

# RHEL 9
daisy -project my-project \
      -zone us-west1-a \
      -var:installer_iso=gs://my-bucket/RHEL9.iso \
      rhel_9.wf.json
```
