## Daisy disk_export workflow
Exports a GCE disk to a GCS location.

Required vars:
+ `source_disk` GCE disk to export
+ `destination` GCS path to export image to

### Command line example
This will export the disk `project/PROJECT/zone/ZONE/disks/MYDISK` to `gs://some/bucket/image.tar.gz`.
```
daisy -project MYPROJECT -zone MYZONE -gcs_path gs://MYBUCKET/daisy/${USERNAME} \
  -variables source_disk=project/MYPROJECT/zone/MYZONE/disks/MYDISK,destination=gs://some/bucket/image.tar.gz \
  disk_export.wf.json
```

### Workflow example
This workflow uses the IncludeWorkflow step to export the disk 
`project/MYPROJECT/zone/ZONE/disks/MYDISK` to `gs://some/bucket/image.tar.gz`.
```json
{
  "Name": "my-workflow",
  "Project": "MYPROJECT",
  "Zone": "MYZONE",
  "GCSPath": "gs://MYBUCKET/daisy/${USERNAME}",
  "Steps": {
    "export-disk": {
      "Timeout": "30m",
      "IncludeWorkflow": {
        "Path": "./disk_export.wf.json",
        "Vars": {
          "source_disk": "project/MYPROJECT/zone/MYZONE/disks/MYDISK",
          "destination": "gs://some/bucket/image.tar.gz"
        }
      }
    }
  }
}
```
