# Debian Image Builds

This directory contains [Daisy] workflows to build the Debian [Public Images]
for GCE.

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Details

Debian is a free operating system and does not require any installation media to
be provided. The mechanism for building depends on the release, and the output
is a GCE Image in the build project.

### Debian 9 (Stretch)

Debian 9 Stretch uses bootstrap-vz from the now archived repo on github.
https://github.com/andsens/bootstrap-vz

Because this is unmaintained, we use a user fork of the repo with our modifications.
https://github.com/hopkiw/bootstrap-vz

### Debian 10 (Buster) and newer

Debian 10 Buster and newer releases use the project FAI (Fully Automated Install) tool, starting from the
officially maintained FAI configs from the Debian cloud images project.
https://salsa.debian.org/cloud-team/debian-cloud-images

The `fai_config` directory contains fai classes and scripts to be used for
testing new Google Cloud packages before they are released in Debian image
builds. There are some additional scripts to turn the Google produced Buster
images into the default baseline GCE experience that the Debian cloud project
does not wish to maintain. The `GCE_SPECIFIC` and `GCE_CLEAN` classes are added
to the upstream config space from the Debian cloud team project.

## Invoking the build workflows

Example Daisy invocations:
```shell

# Debian 9
daisy -project my-project \
      -zone us-west1-a \
      debian_9.wf.json

# Debian 10
daisy -project my-project \
      -zone us-west1-a \
      debian_10_fai.wf.json
```
