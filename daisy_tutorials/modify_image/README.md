# Modifying a base image using Daisy

Daisy workflows [have many
capabilities,](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#workflow-config-overview)
and it is hard to know where to start. This tutorial will demonstrate
writing a very simple workflow to modify a GCE base image.

The overall steps of the workflow will be:

1. Create a new instance
2. Run a provisioning script to modify the instance
3. Create a new disk image from the instance
4. Delete the instance

## Provisioning the instance

What you use to provision the instance is open and not dependent on
Daisy. In this simple case, we are using a simple shell script to
install Emacs. Because we are creating a CentOS image, we simply call
`yum install`. The provisioning script is named `emacs_install.sh`.

```shell
#!/bin/bash

yum -y install emacs-nox
shutdown -h now
```

The script shuts down the instance after it is done. We will use this
later in the Daisy workflow to know when provisioning is done.

## Start writing the workflow

First we will start with a Name:

```
{
    "Name": "emacs-image",
```

Then, list the sources. We will need the provisioning script above.

```
    "Sources": {
        "emacs_install.sh": "./emacs_install.sh"
    },
```

## Create and Provision the Instance

Now we will start writing steps in the workflow. We will create the
disk first, as we need to name it and perform operations on it
later. We give the disk a name, specify the base image, and specify
the SSD type for speed. The name of the step `create-disks` is
arbitrary.

```
    "Steps": {
        "create-disks": {
            "CreateDisks": [
                {
                    "Name": "disk-install",
                    "SourceImage": "projects/centos-cloud/global/images/family/centos-7",
                    "Type": "pd-ssd"
                }
            ]
        },
```

Next create the instance using the above disk, and specify the startup
script that we wrote to provision the instance.

```
        "create-inst-install": {
            "CreateInstances": [
                {
                    "Name": "inst-install",
                    "Disks": [{"Source": "disk-install"}],
                    "MachineType": "n1-standard-1",
                    "StartupScript": "emacs_install.sh"
                }
            ]
        },
```

Lastly, we wait for the provisioning to complete.

```
        "wait-for-inst-install": {
            "TimeOut": "1h",
            "waitForInstancesSignal": [
                {
                    "Name": "inst-install",
                    "Stopped": true
                }
            ]
        },
```

## Capture the Image and Cleanup

Now that the instance provisioning is complete, and the instance is
stopped, we can capture a new image from the disk.

```
        "create-image": {
            "CreateImages": [
                {
                    "Name": "centos-emacs",
                    "SourceDisk": "disk-install",
                    "NoCleanup": true,
                    "ExactName": true
                }
            ]
        },
```

Note the `NoCleanup`, Daisy automatically cleans up any resources
created during a run. We don't want Daisy to delete our image though!
Specifying `NoCleanup` will instruct Daisy to leave the
image. `ExactName` tells Daisy to use the `Name` as-is. Normally Daisy
adds a suffix to resource names to avoid name collisions.

Now we will add a delete step to be explicit.

```
        "delete-inst-install": {
            "DeleteResources": {
                "Instances": ["inst-install"]
            }
        }
    },
```

And that completes the steps for customizing the image!

## Ordering steps.

Daisy will try to run steps in parallel for efficiency, so we need to
specify the order our steps will run. Use the names of the steps.

```
    "Dependencies": {
        "create-inst-install": ["create-disks"],
        "wait-for-inst-install": ["create-inst-install"],
        "create-image": ["wait-for-inst-install"],
        "delete-inst-install": ["create-image"]
    }
}
```

That completes our workflow.

## Running the workflow

Clone the code:

```
git clone https://github.com/GoogleCloudPlatform/compute-image-tools.git
cd compute-image-tools/daisy_tutorials/modify_image/
```

Run Daisy:

```
daisy ./emacs_image.wf.json
```

If you run Daisy from a GCE VM, it will use the project and zone of
the VM. Otherwise specify the `-project` and `-zone` parameters. See
`daisy -h` for details.

If you create your GCE VM to run Daisy from with the `cloud-platform`
scope, it will have full control of your cloud project, and Daisy will
have permissions to perform all the operations. Otherwise use [`gcloud
auth application-default
login`](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/login)
to login with an authorized account.

After the run is complete, check your images:

```shell
$ gcloud compute images list --no-standard-images
NAME                                         PROJECT    FAMILY    DEPRECATED  STATUS
centos-emacs                                 my-project                        READY
```

## Next Steps

Now that you have a basic workflow, you can experiment with the many
options of the steps. Also, using
[variables](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy#vars)
will make the workflow much more flexible.

This workflow is very simple, take a look at the [example
workflows](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy_workflows)
to see full featured automatable workflows.
