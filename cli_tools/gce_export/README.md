## Compute Engine Image Export

The `gce_export` tool streams a local disk to a Google Compute Engine image 
file in a Google Cloud Storage bucket (steps 9 and 10 in the
[image export documentation](https://cloud.google.com/compute/docs/images/export-image)).
When exporting to GCS, no local file is created so no additional disk space needs to be allocated. 
Once complete the image file can be imported in GCE as described in the 
[image import documentation](https://cloud.google.com/compute/docs/images/import-existing-image).

This tool can also be used to export the image to local disk.

### Build
Download and install [Go](https://golang.org/doc/install). Then pull and 
install the `gce_export` tool, this should place the binary in the 
[Go bin directory](https://golang.org/doc/code.html#GOPATH):

```
go get github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_export
```

### Flags

+ `-disk` disk to export, on linux this would be something like `/dev/sdb`, and on
Windows `\\.\PhysicalDrive1`
+ `-buffer_prefix` if set will use this local path as the local buffer prefix, on linux this would
be something like `/my_folder`, and on Windows `\\.\PhysicalDrive2\path`.
+ `-gcs_path` GCS path to upload the image to, in the form of gs://my-bucket/image.tar.gz
+ `-oauth` path to oauth json file for authenticating to the GCS bucket
+ `-licenses` (optional) comma separated list of licenses to add to the image
+ `-y` skip confirmation prompt

### Usage

While you can export a disk with currently mounted partitions, or even the disk
containing the current root partition it is recommended to unmount all partitions
prior to running `gce_export`.

#### Linux:

This will stream `/dev/sdb` to the GCS path gs://some-bucket/linux.tar.gz

```
gce_export -disk /dev/sdb -gcs_path gs://some-bucket/linux.tar.gz
```

This will stream `/dev/sdb` to the local path /my_folder/linux.tar.gz

```
gce_export -disk /dev/sdb -local_path /my_folder/linux.tar.gz
```

#### Windows:

This will stream `\\.\PhysicalDrive1` to the GCS path
gs://some-bucket/path/windows.tar.gz

```
gce_export.exe -disk \\.\PhysicalDrive1 -gcs_path gs://some-bucket/windows.tar.gz
```

This will stream `\\.\PhysicalDrive1` to the local path
`\\.\PhysicalDrive2\path\windows.tar.gz`

```
gce_export.exe -disk \\.\PhysicalDrive1 -local_path \\.\PhysicalDrive2\path\windows.tar.gz
```

