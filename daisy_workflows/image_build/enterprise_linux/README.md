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

1. Create an instance using the `debian-12-worker` image for the boot disk, an
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
content; use of this content is provided as part of your GCE customer agreement
and is billed based on instance usage.

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

## Updating RHEL build workflows

All of the RHEL image_build workflow files should not be edited manually. They
should only be managed by editing & running the [write_image_build_workflow.py](https://github.com/GoogleCloudPlatform/compute-image-tools/blob/master/daisy_workflows/image_build/enterprise_linux/write_image_build_workflow.py)
file.

### Adding new major release (ex. "RHEL 10")

1. Write a new consolidated kickstart file based on major release versions (ex. "rhel_10_consolidated.cfg")

1. Add the new major release number to the `RHEL_MAJOR_VERSIONS` list at the top of write_image_build_workflow.py

1. Update the get_guest_os_features function in the write_image_build_workflow.py script & any other applicable
   new changes to the image build files

1. Run the following command to generate the new workflow files
   ```bash
   python3 /daisy_workflows/image_build/enterprise_linux/write_image_build_workflow.py
   ```

1. Send out the script changes & the new workflow files for review as a PR.

### Adding new RHEL Variant to major release (ex. "RHEL 10.0 for SAP")

1. Add new minor release number to the respective variant list (EUS/LVM/SAP)

1. Update the major release's consolidated kickstart file with the necessary variant specific changes

1. Run the following command to generate the new workflow files
   ```bash
   python3 /daisy_workflows/image_build/enterprise_linux/write_image_build_workflow.py
   ```

1. Send out the script changes & the new workflow files for review as a PR.

### Adding new minor point release (ex. "RHEL 9.6 for SAP")

1. Add the new minor release version (ex. "9.6") to the appropriate list at the
   top of the file.

1. Run the following command to generate the new workflow files
   ```bash
   python3 /daisy_workflows/image_build/enterprise_linux/write_image_build_workflow.py
   ```

1. Send out the script changes & the new workflow files for review as a PR.
