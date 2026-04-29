# Daisy Workflows

These are [Daisy] workflows used by Google, primarily to build the [Public Images]

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Directory layout

The following directories exist:

* image\_build - base workflows used to build Public Images
* build-publish - workflows used by Google, for both private and public images
* export - workflows for [Export Image] to GCS
* export\_metadata - workflows for automating post-build image scanning
* linux\_common - the bootstrap shell scripts and python libraries used by many
  workflows
* ovf\_import - workflows for [OVF Import]

[Export Image]: https://cloud.google.com/compute/docs/images/export-image#export_an_image_to
[OVF Import]: https://cloud.google.com/compute/docs/import/import-ovf-files
