# Debian Image Builds
Example Daisy invocations:
```shell

# Debian 8 (using gcloud application-default credentials)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -variables image_dest=gs://bucket/images/debian_8 \
      debian8.wf.json


# Debian 9 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables image_dest=gs://bucket/images/debian_9 \
      debian9.wf.json
```

The `google_cloud_test_repos` directory is a bootstrap-vz plugin to be used for
testing new Google Cloud packages before they are released in Debian image
builds. It is therefore not included in bootstrap-vz.
