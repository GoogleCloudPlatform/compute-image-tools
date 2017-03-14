{
  "steps": {
    "create disks": {
      "create_disks": [
        {
          "name": "bootstrap",
          "source_image": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50",
          "ssd": true
        }
      ]
    },
    "bootstrap": {
      "create_instances": [
        {
          "name": "bootstrap",
          "attached_disks": ["bootstrap"],
          "metadata": {
            "test_metadata": "this was a test"
          },
          "machine_type": "n1-standard-1",
          "startup_script": "shutdown /h"
        }
      ]
    },
    "bootstrap stopped": {
      "timeout": "1h",
      "wait_for_instances_stopped": ["bootstrap"]
    }
  },
  "dependencies": {
    "bootstrap": ["create disks"],
    "bootstrap stopped": ["bootstrap"]
  }
}
