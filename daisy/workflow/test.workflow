{
  "name": "some-name",
  "project": "some-project",
  "zone": "us-central1-a",
  "gcsPath": "gs://some-bucket/images",
  "oauthPath": "somefile",
  "vars": {
    "bootstrap_instance_name": {"Value": "bootstrap", "Required": true},
    "machine_type": "n1-standard-1"
  },
  "steps": {
    "create-disks": {
      "createDisks": [
        {
          "name": "bootstrap",
          "sourceImage": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "Type": "pd-ssd"
        },
        {
          "name": "image",
          "sourceImage": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "Type": "pd-standard"
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
    "${bootstrap_instance_name}-stopped": {
      "timeout": "1h",
      "waitForInstancesSignal": [{"name": "${bootstrap_instance_name}", "stopped": true, "interval": "1s"}]
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
    "postinstall-stopped": {
      "waitForInstancesSignal": [{"name": "postinstall", "stopped": true}]
    },
    "create-image": {
      "createImages": [
        {
          "name": "image-from-disk",
          "sourceDisk": "image"
        }
      ]
    },
    "merge-workflow": {
      "MergeWorkflow": {
        "path": "./test_sub.workflow"
      }
    },
    "sub-workflow": {
      "subWorkflow": {
        "path": "./test_sub.workflow"
      }
    }
  },
  "dependencies": {
    "create-disks": [],
    "bootstrap": ["create-disks"],
    "bootstrap-stopped": ["bootstrap"],
    "postinstall": ["bootstrap-stopped"],
    "postinstall-stopped": ["postinstall"],
    "create-image": ["postinstall-stopped"],
    "merge-workflow": ["create-image"],
    "sub-workflow": ["create-image"]
  }
}
