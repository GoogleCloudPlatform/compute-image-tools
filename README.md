# Compute Engine Image Tools
[![codecov](https://codecov.io/gh/GoogleCloudPlatform/compute-image-tools/branch/master/graph/badge.svg)](https://codecov.io/gh/GoogleCloudPlatform/compute-image-tools)

This repository contains various tools for managing disk images on Google
Compute Engine. A description of each tool and directory is below

## [Daisy](daisy)

Daisy is a solution for running complex, multi-step workflows on GCE.

### [Daisy Workflows](daisy_workflows)

Full featured Daisy workflow examples, image builds, and image import
workflows. A [user guide](daisy_workflows/import_userguide.md) for VM imports is
also provided here.

### [Daisy Tutorials](daisy_tutorials)

Basic workflow examples and tutorials for getting started with Daisy.

## [GCE Export tool](gce_export)

The gce_export tool streams a local disk to a Google Compute Engine image file
in a Google Cloud Storage bucket.

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
