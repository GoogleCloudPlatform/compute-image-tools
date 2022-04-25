# Compute Image Tools

Tools for building, testing, releasing, and upgrading
[Google Compute Engine images](https://cloud.google.com/compute/docs/images).

## [GCE Export](cli_tools/gce_export)

Streams an attached Google Compute Engine disk to an image file in a Google
Cloud Storage bucket.

**Docker**

- Latest: `gcr.io/compute-image-tools/gce_export:latest`
- Release: `gcr.io/compute-image-tools/gce_export:release`

**Linux x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/linux/gce_export)
- [Release](https://storage.googleapis.com/compute-image-tools/release/linux/gce_export)

**Windows x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/windows/gce_export.exe)
- [Release](https://storage.googleapis.com/compute-image-tools/release/windows/gce_export.exe)

## [Windows Upgrade](cli_tools/gce_windows_upgrade)

Performs in-place OS upgrades. The tool can be invoked
with [`gcloud beta compute os-config os-upgrade`](https://cloud.google.com/sdk/gcloud/reference/beta/compute/os-config/os-upgrade).


**Docker**

- Latest: `gcr.io/compute-image-tools/gce_windows_upgrade:latest`
- Release: `gcr.io/compute-image-tools/gce_windows_upgrade:release`

**Linux x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/linux/gce_windows_upgrade)
- [Release](https://storage.googleapis.com/compute-image-tools/release/linux/gce_windows_upgrade)


## [Image Publish](cli_tools/gce_image_publish)

Creates Google Compute Engine images from raw disk files.

**Docker**

- Latest: `gcr.io/compute-image-tools/gce_image_publish:latest`
- Release: `gcr.io/compute-image-tools/gce_image_publish:release`

**Linux x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/linux/gce_image_publish)
- [Release](https://storage.googleapis.com/compute-image-tools/release/linux/gce_image_publish)

**Windows x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/windows/gce_image_publish.exe)
- [Release](https://storage.googleapis.com/compute-image-tools/release/windows/gce_image_publish.exe)

**OSX x64**

- [Latest](https://storage.googleapis.com/compute-image-tools/latest/darwin/gce_image_publish)
- [Release](https://storage.googleapis.com/compute-image-tools/release/darwin/gce_image_publish)



## Contributing

Have a patch that will benefit this project? Awesome! Follow these steps to have
it accepted.

1.  Please sign our [Contributor License Agreement](CONTRIBUTING.md).
1.  Fork this Git repository and make your changes.
1.  Create a Pull Request.
1.  Incorporate review feedback to your changes.
1.  Accepted!

## License

All files in this repository are under the
[Apache License, Version 2.0](LICENSE) unless noted otherwise.
