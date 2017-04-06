# Daisy

Daisy is a GCE workflow tool. Development is ongoing.
https://godoc.org/github.com/GoogleCloudPlatform/compute-image-tools/daisy

## Table of contents

  * [Workflow Sources](#workflow-sources)
  * [Workflow Steps](#workflow-steps)
    * [AttachDisks](#attachdisks)
    * [CreateDisks](#createdisks)
    * [CreateImages](#createimages)
    * [CreateInstances](#createinstances)
    * [DeleteResources](#deleteresources)
    * [RunTests](#runtests)
    * [SubWorkflow](#subworkflow)
    * [WaitForInstancesSignal](#waitforinstancessignal)
    * [WaitForInstancesStopped](#waitforinstancesstopped)
  * [Dependency Map](#dependency-map)

## Workflow Sources

Daisy will upload any workflow sources to the sources directory in GCS
prior to running the workflow. The `sources` field in a workflow
JSON file is a map of 'destination' to 'sources'. Sources can be a local
or GCS file or folder. Folders will be recursively copied into
destination.

In this example the local file `./path/to/startup.sh` will be copied to
`startup.sh` in the sources folder. Similarly the GCS file
`gs://my-bucket/some/path/install.py` will be copied to `install.py`.
The contents of paths referencing directories like
`./path/to/drivers_folder` and  `gs://my-bucket/my-files` will be
recursively copied to the directories `drivers` and `files` in GCS
respectively.

```
"sources": {
  "startup.sh": "./path/to/startup.sh",
  "install.py": "gs://my-bucket/some/path/install.py",
  "drivers": "./path/to/drivers_folder",
  "files": "gs://my-bucket/my-files"
}
```

## Workflow Steps
Step types are defined here:
https://godoc.org/github.com/GoogleCloudPlatform/compute-image-tools/daisy/workflow#Step

In a workflow file the `steps` field is a mapping of step names to their
type descriptions. The name can be whatever you choose, it's how you
will reference the steps in the dependency map as well as how they will
show up in the logs. For each individual 'step' you set one 'step type'
along with any of its required fields.
```
"steps": {
  "step name 1" {
    "stepType" {
      ...
    }
  },
  "step name 2" {
    "stepType" {
      ...
    }
  }
}
```

### AttachDisks
Not implemented yet.

### CreateDisks
Creates GCE disks.

This CreateDisks step example creates two disks: the first is a standard
PD disk created from a source image, the second is blank PD SSD.
```
"create disks step": {
  "createDisks": [
    {
      "name": "disk1",
      "sourceImage": "projects/debian-cloud/global/images/family/debian-8"
    },
    {
      "name": "disk2",
      "sizeGb": "200",
      "type": "pd-ssd"
    }
  ]
}
```

### CreateImages
Creates GCE images.

This CreateImages example creates an image from a source disk.
```
"create image step": {
  "createImages": [
    {
      "name": "image1",
      "sourceDisk": "disk2"
    }
  ]
}
```

This CreateImages example creates an image from a file in GCS, it also
uses the no_cleanup flag to tell Daisy that this resource should exist
after workflow completion, and the exact_name flag to tell Daisy to not
use an generated name for the resource.
```
"create image step": {
  "createImages": [
    {
      "name": "image1",
      "sourceFile": "gs://my-bucket/image.tar.gz",
      "noCleanup": true,
      "exactName": true
    }
  ]
}
```

### CreateInstances
Creates GCE instances.

This CreateInstances step example creates an instance with two attached
disks and uses the machine type n1-standard-4.
```
"create instances step": {
  "createInstances": [
    {
      "name": "instance1",
      "attachedDisks": ["disk1", "disk2"],
      "machineType": "n1-standard-4"
    }
  ]
}
```

### DeleteResources
Deletes GCE resources (images, instances, disks). Any disks listed will
be deleted after any listed instances.

This DeleteResources step example deletes an image, an instance, and two
disks.
```
"delete resources step": {
  "deleteResources": {
     "images":["image1"],
     "instances":["instance1"],
     "disks":["disk1", "disk2"]
   }
}
```

### RunTests
Not implemented yet.

### SubWorkflow
Runs a Daisy subworkflow.

This SubWorkflow step example uses a local workflow file.
```
"sub workflow step": {
  "subWorkflow": {
    "path": "./some_subworkflow.workflow"
  }
}
```

### WaitForInstancesSignal
Not implemented yet.

### WaitForInstancesStopped
Waits for a set of instances to stop.

This WaitForInstancesStopped step example waits up to 1 hour for
'instance1' to stop.
```
"wait for instances stopped step": {
  "timeout": "1h",
  "waitForInstancesStopped": ["instance1"]
},
```

## Dependency Map

The dependency map describes the order in which workflow steps will run.
Steps without any dependencies will run immediately, otherwise a step
will only run once its dependencies have completed successfully.

In this example step1 will run immediately as it has no dependencies,
step2 and step3 will run as soon as step1 completes, and step4 will run
as soon as both step2 and step3 complete.
```
"steps": {
  "step1" {
    ...
  },
  "step2" {
    ...
  },
  "step3" {
    ...
  },
  "step4" {
    ...
  }
}
"dependencies": {
  "step2": ["step1"],
  "step3": ["step1"],
  "step4": ["step2", "step3"]
}
```