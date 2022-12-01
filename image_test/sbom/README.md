These tests validate sbom generation. 

This is an example command to test if the SBOM is correctly generated for almalinux 9 images. The raw disk
tar.gz file and sbom json file are expected to appear in a scratch path within the input Google Cloud Project.

```
daisy -project {project name} -var:syft_source=path_to_syft_source enterprise_sbom_test.wf.json
```


