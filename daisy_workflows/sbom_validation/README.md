These tests validate sbom generation. 

This is an example command to test if the SBOM is correctly generated for almalinux 9 images. The raw disk
tar.gz file and sbom json file are expected to appear in a scratch path within the input Google Cloud Project.

```
daisy -project {project name} -var:sbom_util_gcs_root=root_of_gcs_filesystem enterprise_sbom_test.wf.json
```

The root of sbom util should point to a bucket including folders such as "linux", "linux_arm64", and "windows", corresponding to the operating systems of the images which we want to generate the SBOM for.
