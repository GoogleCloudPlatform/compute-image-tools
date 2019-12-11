# Daisy image/disk export workflows
Exports a GCE disk to a GCS location.

There are two types of export worklfows depending on the required output 
format. 

## GCE raw disk image
`image_export.wf.json` and `disk_export.wf.json` export a raw image in
a tar.gz, this is the 'native' GCE format and the resulting image can 
be imported directly to GCE.

Required vars:
+ `source_image` GCE image to export
+ `destination` GCS path to export image to

### Command line example
This will export the disk `project/PROJECT/gloabl/images/MYIMAGE` to `gs://some/bucket/image.tar.gz`.
```
daisy -project MYPROJECT -zone MYZONE \
  -var:source_image=project/PROJECT/gloabl/images/MYIMAGE
  -var:destination=gs://some/bucket/image.tar.gz \
  image_export.wf.json
```

### Workflow example
This workflow uses the IncludeWorkflow step to export the disk 
`project/PROJECT/gloabl/images/MYIMAGE` to `gs://some/bucket/image.tar.gz`.
```json
{
  "Name": "my-workflow",
  "Project": "MYPROJECT",
  "Zone": "MYZONE",
  "GCSPath": "gs://MYBUCKET/daisy/${USERNAME}",
  "Steps": {
    "export-image": {
      "Timeout": "30m",
      "IncludeWorkflow": {
        "Path": "./image_export.wf.json",
        "Vars": {
          "source_image": "project/PROJECT/gloabl/images/MYIMAGE",
          "destination": "gs://some/bucket/image.tar.gz"
        }
      }
    }
  }
}
```

## Alternate disk image formats
`image_export_ext.wf.json` and `disk_export_ext.wf.json` allow the specifying 
of common image formats for the output image.

We use [qemu-img](http://www.qemu.org/documentation) to do the conversion. 
Valid output formats are:

* raw
* qcow2
* qcow
* vmdk
* vdi
* vhdx
* qed
* vpc

Required vars:
+ `source_image` GCE image to export
+ `destination` GCS path to export image to
+ `format` Format for the exported image

### Command line example
This will export the disk `project/PROJECT/gloabl/images/MYIMAGE` to `gs://some/bucket/image.vmdk`.
```
daisy -project MYPROJECT -zone MYZONE \
  -var:source_image=project/PROJECT/gloabl/images/MYIMAGE
  -var:destination=gs://some/bucket/image.vmdk \
  -var:format=vmdk \
  image_export_ext.wf.json