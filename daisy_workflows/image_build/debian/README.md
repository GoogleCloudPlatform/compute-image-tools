# Debian Image Builds

Debian 9 Stretch uses bootstrap-vz from the now archived repo on github.
https://github.com/andsens/bootstrap-vz

Going forward, Debian 10 Buster and future releases will use the project Fai
configs from the Debian cloud images project.
https://salsa.debian.org/cloud-team/debian-cloud-images

Example Daisy invocations:
```shell

# Debian 9 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables image_dest=gs://bucket/images/debian_9.tar.gz \
      debian_9.wf.json

# Debian 10 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables image_dest=gs://bucket/images/debian_10.tar.gz \
      debian_10_fai.wf.json
```

The `google_cloud_test_repos` directory is a bootstrap-vz plugin to be used for
testing new Google Cloud packages before they are released in Debian image
builds. It is therefore not included in bootstrap-vz.

The `fai_config` directory contains fai classes and scripts to be used for
testing new Google Cloud packages before they are released in Debian image
builds. There are some additional scripts to turn the Google produced Buster
images into the default baseline GCE experience that the Debian cloud project
does not wish to maintain. The `GCE_SPECIFIC` and `GCE_CLEAN` classes are added
to the upstream config space from the Debian cloud team project.
