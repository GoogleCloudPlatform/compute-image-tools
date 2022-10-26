# Daisy image build workflows (Google use)

These are [Daisy] and [`gce_image_publish`] workflows used by Google to build
[Public Images] for GCE and some images used by internal teams.

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[`gce_image_publish`]: https://github.com/GoogleCloudPlatform/compute-image-tools/#image-publish
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Directory layout

The following directories exist

* bare\_metal - wrapper workflows for [bare\_metal](../image_build/bare-metal)
* debian - wrapper workflows for [debian](../image_build/debian)
* enterprise\_linux - wrapper workflows for
  [enterprise\_linux](../image_build/enterprise_linux)
* kokoro - workflow for an internal use image
* linux\_dev - unknown/unused (?)
* rhui - workflows for internal use images
* sqlserver - wrapper workflows for [sqlserver](../image_build/sqlserver)
* windows - wrapper workflows for [windows](../image_build/windows)
* windows\_container - wrapper workflows for
  [windows\_container](../image_build/sqlserver)

## Standards

The image build workflows below this directory most often produce a GCS tarball
in the format suitable for use in creating a GCE Image.

Most workflows below this directory are 'wrappers' around their counterparts in
the `image_build` directory. These may contain additional hardcoded parameters
specific to the way Google builds and publishes the images, or simply wrap them
in order to produce a GCS tarball rather than a GCE Image directly. Google's
CI/CD systems create many GCE Images from a single GCS tarball in order to
obtain bit-for-bit equality without necessarily establishing a parent/child
relationship between images.
