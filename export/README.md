## Compute Engine Image export

The export tool streams a local disk to a Google Compute Engine image file in
a Google Cloud Storage bucket (steps 9 and 10 here:
https://cloud.google.com/compute/docs/images/export-image). No local file is
created so no additional disk space needs to be allocated. Once complete the
image file can be imported in GCE as described here:
https://cloud.google.com/compute/docs/images/import-existing-image

### Flags

+ -disk: disk to copy, on linux this would be something like '/dev/sdb', and on
Windows '\\.\PhysicalDrive1'
+ -bucket: GCS bucket to copy the image to
+ -out: (optional) what to call the resultant image file (.tar.gz will be appened)
+ -licenses: (optional) comma deliminated list of licenses to add to the image
+ -y: skip confirmation prompt

### Usage

While you can export a disk with currently mounted partitions, or even the disk
containing the current root partition it is recommened to unmount all partitions
prior to running export.

#### Linux:

This will stream /dev/sdb to the GCS path gs://some-bucket/linux.tar.gz

```
export -bucket some-bucket -disk /dev/sdb -out linux
```

#### Windows:

This will stream '\\.\PhysicalDrive1' to the GCS path
gs://some-bucket/windows.tar.gz

```
export.exe -bucket some-bucket -disk \\.\PhysicalDrive1 -out windows
```

