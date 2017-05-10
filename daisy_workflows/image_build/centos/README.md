# CentOS Image Builds
Example Daisy invocations:
```shell

# CentOS 6 (using gcloud application-default credentials)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -variables \
              image-dest=gs://bucket/images,\
              installer-iso=co6_installer.iso \
      centos/centos6.wf.json


# CentOS 7 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables \
              image-dest=gs://bucket/images,\
              installer-iso=co7_installer.iso \
      centos/centos7.wf.json
```