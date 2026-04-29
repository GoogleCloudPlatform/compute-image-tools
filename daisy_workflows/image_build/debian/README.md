# Debian Image Builds

This directory contains [Daisy] workflows to build the Debian [Public Images]
for GCE.

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Details

Debian is a free operating system and does not require any installation media to
be provided. The mechanism for building depends on the release, and the output
is a GCE Image in the build project.

Debian build workflows all follow the steps:

1. Create an instance using the `debian-11-worker` image as a boot disk and the
   appropriate installer script as startup script

1. On boot, the startup script performs the relevant build steps which output a
   raw disk image, then uploads it to GCS.

1. The file in GCS is then used to create a GCE Image.

### Debian build process

Debian releases use the FAI (Fully Automated Install) tool, starting from the
officially maintained FAI configs from the Debian cloud images project. This
build doesn't use the full toolset provided by the cloud images team project.

https://salsa.debian.org/cloud-team/debian-cloud-images

The `fai_config` directory contains FAI classes and scripts which deviate from
the base Debian cloud images team build in order to support the GCE guest
envirnoment on Debian. Therefore the final config used is a mix of the upstream
classes and the ones in the `fai_config` directory layered in.

## Invoking the build workflows

Example Daisy invocations:
```shell

# Debian 11
daisy -project my-project \
      -zone us-west1-a \
      debian_11_fai.wf.json
```
