# Debian Image Builds
Example Daisy invocations:
```shell

# Debian 8 (using gcloud application-default credentials)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -variables image_dest=gs://bucket/images \
      debian8.wf.json


# Debian 9 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables image_dest=gs://bucket/images \
      debian9.wf.json
```
