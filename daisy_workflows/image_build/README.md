# Daisy image build workflows

These are [Daisy] workflows used by Google to build [Public Images] for GCE.

[Daisy]: https://github.com/GoogleCloudPlatform/compute-daisy/tree/master/docs
[Public Images]: https://cloud.google.com/compute/docs/images#os-compute-support

## Directory layout

The following directories exist (see relevant READMEs for more details):

* [debian](debian/README.md) - workflows to build Debian images
* [enterprise\_linux](enterprise_linux/README.md) - workflows to build
  Enterprise Linux images
* install\_package - derivative workflow to install a package onto an image
* [sqlserver](sqlserver/README.md) - derivative workflow to install MSSQL Server
  onto Windows images
* [windows](windows/README.md) - workflows to build Windows images
* windows\_for\_containers - derivative workflows to install Docker EE onto
  Windows images

Note: At one point, [Rocky Linux](https://rockylinux.org/)
builds were nested under `enterprise_linux` but have since been taken over by
[CIQ](https://ciq.com/), which can be found
[here](https://github.com/ctrliq/gcp-public-images/tree/main/daisy).

## Standards

The image build workflows below this directory most often produce a [GCE Image].

[GCE Image]: https://cloud.google.com/compute/docs/reference/rest/v1/images

It is a common practice to use a common 'base' workflow which is run in more
specific workflows using Daisy's `IncludeWorkflow` directive. Doing so requires
the including workflow to specify all required parameters in the base workflow.
The more specific workflows often exist to codify the necessary parameters for
the base workflow, but don't otherwise alter the method in which it operates.

It is a common practice to use the bootstrap shell script located in the
[linux\_common](x) directory to prepare a VM for using python scripts. This
eliminates the need for scripts which have to bootstrap themselves.

It is a common practice to use a pre-prepared 'worker' image, most often the
Debian 12 worker image, which is publicly available in the
`compute-image-tools` GCP project. The workflows for building these worker
images are also present in the `debian/` directory.
