# Image Imports
Example Daisy invocation:
```shell
# Example importing a VMDK (using a credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json
      -variables \
              source=image.vmdk,\
              image-name=my-new-imported-image \
      imageimport.wf.json
```