# Automating Image Creation with Daisy

This page describes how to use Daisy to automate creation of custom GCE images.
These images are usually based on an existing base image (often a fresh install
of your OS of choice), with some extra software you have chosen to install. This
page assumes you have an existing workflow for creating such an image, and
describes how to use Daisy to automate this task.

To begin using Daisy for your image creation workflows, follow these steps:

  1. Choose a base image to start with.
  2. Enumerate all the steps you currently follow to prepare your image.
  3. Make sure all those steps can be done from the command line.
  4. Write a new (or edit an existing) script to run on your base image which
     does *all* of the steps you identified in step 2.
  5. Write a Daisy workflow which creates at least one disk (the first disk
     should be your base image), starts an instance from that disk, runs your
     script, waits for the script to finish, then takes an image from the disk.

The rest of this page describes these steps in greater depth.

## Selecting a Base Image

Your base image can come from anywhere, but if you're starting with an OS that
is available on Compute Engine, then you can use a "partial URL" to get it
directly from Google rather than uploading your own image.

Additionally, to avoid pinning your script to a specific release, you can use
the image "family" as a partial URL, rather than naming a specific image.
Partial URLs for image families have the following form:

    projects/<project>/global/images/family/<family-name>

To learn more about the `sourceImage` property and partial URLs, see the
["Disks"
page](https://cloud.google.com/compute/docs/reference/latest/disks#sourceImage)
of the Compute Engine API docs.

You will need to find the correct project and family name for your chosen image
family. To do this, navigate in the Cloud Console to Compute Engine -> Images,
then click on the name of the instance you would like to use. On this page will
be listed the image's family. To find the project, click on "Equivalent REST"
and find the "selfLink" field. The second part of the path is the project name.
Note that the project name is not the name of your own project, it is the name
of the project which contains your chosen base image. For example, the partial
URL for the Ubuntu 16.04 LTS image family is:

    projects/ubuntu-os-cloud/global/images/family/ubuntu-1604-lts

You can use this partial URL in the "SourceImage" field in a "CreateDisks" Daisy
step, or anywhere else that Daisy accepts partial URLs. By using an image
family, you ensure that you will always run your workflow on the most up-to-date
base image available.

## Automating your Workflow

We recommend that you write down every step you perform to prepare your image,
in the order they are performed. This will be different for every image, but it
usually involves some combination of downloading and running installers,
building code, invoking package managers, and writing to configuration files.

The biggest thing to keep in mind is that Daisy only allows you to run one
script on startup. This script can download and run other scripts, but it must
be the one to do it. Once you have listed all of the steps you take to set up
your image, write a script that performs all of them.

All of the files that your startup script needs to access (which are not already
present on the machine) should be listed in the "Sources" field of your Daisy
workflow. Sources can be paths to local files, or to objects in a Storage bucket
using the URL format `gs://<bucket>/<file>`.

Files listed in the "Sources" field of your Daisy workflow are all copied to a
scratch directory for your workflow in Google Cloud Storage. For more
information on getting the URL of this location and downloading files from it,
see [Passing Data to Instances](daisy-passing-data.md).

All Windows image creation startup scripts should end with a call to
`gcesysprep.bat`. All other OS scripts should end by shutting down the instance.

## A Typical Workflow

The typical Daisy image creation workflow has five steps:

1. A CreateDisks step which creates at least one disk. The SourceImage for one
   of these disks should be your base image.
2. A CreateInstances step which creates an instance out of the disks the
   previous step created. The first disk in this step's `Disks` field must be
   bootable. This step also specifies the StartupScript.
3. A WaitForInstancesSignal step which waits for the startup script to complete.
   WaitForInstancesSignal has a field called `Stopped` to listen for the
   instance shutting down. It can also be made to listen for specific output on
   the serial port.
4. A CreateImages step which actually creates an image from the disk once the
   instance has shut down. Make sure you specify `NoCleanup` on this step, or
   your image will be deleted when the workflow ends!
5. A DeleteResources step that deletes the instance you created in the
   CreateInstances step.

Each step should depend on the step before it. Use the `Dependencies` field of
your Daisy workflow to specify dependencies. If no dependencies are specified,
then Daisy will try to run all the steps in parallel. Daisy will produce an
error if a step tries to use a resource from another step it does not depend on.

For an example of a typical Daisy image creation workflow, please see the [SQL
Server example
workflow](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy_workflows/image_build/sqlserver).

## Making Multiple Images

If you have many similar images you would like to create (e.g. both Ubuntu 14.04
LTS and Ubuntu 16.04 LTS, with the same extra software installed), you can put
the shared parts of the workflow in one common workflow file, and then create a
small(er) workflow file for each variant image you need to create. For an
example of this, see the [Debian example
workflows](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy_workflows/image_build/debian).
Documentation on using multiple workflow files can be found
[here](daisy-reusing-workflows.md).
