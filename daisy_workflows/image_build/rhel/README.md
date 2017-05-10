# RHEL Image Builds
Example Daisy invocations (this does not work externally from Google... yet):
```shell

# RHEL 6 (using gcloud application-default credentials)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -variables \
              image-dest=gs://bucket/images,\
              installer-iso=rhel6_installer.iso,\
              rhui-client-rpm=rhel6_rhui_client.rpm \
      rhel6.wf.json


# RHEL 7 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables \
              image-dest=gs://bucket/images,\
              installer-iso=rhel7_installer.iso,\
              rhui-client-rpm=rhel7_rhui_client.rpm \
      rhel7.wf.json
```