# Enterprise Linux Image Builds

CentOS builds will work for anyone.

Red Hat builds with these workflows currently require access to the GCE RHUI
packages to enable Red Hat repositories. A variant of this build can build Red
Hat images without RHUI and with Red Hat subscriptions instead. In that case,
you would also remove the GCE license from the resulting image.

Example Daisy invocations:
```shell

# CentOS 7 (using credentials file)
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables \
              installer_iso=centos7_installer.iso \
      centos_7.wf.json

# RHEL 7
daisy -project my-project \
      -zone us-west1-a \
      -gcs_path gs://bucket/daisyscratch \
      -oauth creds.json \
      -variables \
             installer_iso=rhel7_installer.iso,rhui_client_rpm=gce_rhui.rpm \
      rhel_7.wf.json
```
