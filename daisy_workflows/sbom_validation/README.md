These tests validate sbom generation. The workflow prints whether the SBOM generation succeeded or failed to the command line, and then deletes the sbom and exported image file. This test is also used as a presubmit whenever daisy_workflows/export/export_disk.sh is modified, and should take around 5-7 minutes to run.

This is an example command to test if the SBOM is correctly generated for centos 7 images, though the source image being tested can be changed in the setup-disks step of enterprise_sbom_test.wf.json.

```
daisy -project {project name} -var:sbom_util_gcs_root=root_of_gcs_filesystem enterprise_sbom_test.wf.json
```

The root of sbom util should point to a bucket including folders such as "linux", "linux_arm64", and "windows", corresponding to the operating systems of the images which we want to generate the SBOM for. 
