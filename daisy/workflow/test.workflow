{
  "name": "some-name",
  "project": "some-project",
  "zone": "us-central1-a",
  "gcsPath": "gs://some-bucket/images",
  "vars": {
    "bootstrap_instance_name": "bootstrap",
    "machine_type": "n1-standard-1"
  },
  "steps": {
    "create disks": {
      "createDisks": [
        {
          "name": "bootstrap",
          "sourceImage": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "DiskType": "pd-ssd"
        },
        {
          "name": "image",
          "sourceImage": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "DiskType": "pd-standard"
        }
      ]
    },
    "${bootstrap_instance_name}": {
      "createInstances": [
        {
          "name": "${bootstrap_instance_name}",
          "attachedDisks": ["bootstrap", "image"],
          "metadata": {
            "test_metadata": "this was a test"
          },
          "machineType": "${machine_type}",
          "startupScript": "shutdown /h"
        }
      ]
    },
    "${bootstrap_instance_name} stopped": {
      "timeout": "1h",
      "waitForInstancesStopped": ["${bootstrap_instance_name}"]
    },
    "postinstall": {
      "createInstances": [
        {
          "name": "postinstall",
          "attachedDisks": ["image", "bootstrap"],
          "machineType": "${machine_type}",
          "startupScript": "shutdown /h"
        }
      ]
    },
    "postinstall stopped": {
      "waitForInstancesStopped": ["postinstall"]
    },
    "create image": {
      "createImages": [
        {
          "name": "image-from-disk",
          "sourceDisk": "image"
        }
      ]
    },
    "sub workflow": {
      "subWorkflow": {
        "path": "./test_sub.workflow"
      }
    }
  },
  "dependencies": {
    "create disks": [],
    "bootstrap": ["create disks"],
    "bootstrap stopped": ["bootstrap"],
    "postinstall": ["bootstrap stopped"],
    "postinstall stopped": ["postinstall"],
    "create image": ["postinstall stopped"],
    "sub workflow": ["create image"]
  }
}
