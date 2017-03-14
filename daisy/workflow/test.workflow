{
  "name": "some-name",
  "project": "some-project",
  "zone": "us-central1-a",
  "bucket": "some-bucket/images",
  "vars": {
    "bootstrap_instance_name": "bootstrap",
    "machine_type": "n1-standard-1"
  },
  "steps": {
    "create disks": {
      "create_disks": [
        {
          "name": "bootstrap",
          "source_image": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "ssd": true
        },
        {
          "name": "image",
          "source_image": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "ssd": true
        }
      ]
    },
    "${bootstrap_instance_name}": {
      "create_instances": [
        {
          "name": "${bootstrap_instance_name}",
          "attached_disks": ["bootstrap", "image"],
          "metadata": {
            "test_metadata": "this was a test"
          },
          "machine_type": "${machine_type}",
          "startup_script": "shutdown /h"
        }
      ]
    },
    "${bootstrap_instance_name} stopped": {
      "timeout": "1h",
      "wait_for_instances_stopped": ["${bootstrap_instance_name}"]
    },
    "postinstall": {
      "create_instances": [
        {
          "name": "postinstall",
          "attached_disks": ["image", "bootstrap"],
          "machine_type": "${machine_type}",
          "startup_script": "shutdown /h"
        }
      ]
    },
    "postinstall stopped": {
      "wait_for_instances_stopped": ["postinstall"]
    },
    "create image": {
      "create_images": [
        {
          "name": "image-from-disk",
          "source_disk": "image"
        }
      ]
    },
    "sub workflow": {
      "sub_workflow": {
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
