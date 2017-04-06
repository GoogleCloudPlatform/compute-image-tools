{
  "steps": {
    "create disks": {
      "createDisks": [
        {
          "name": "bootstrap",
          "sourceImage": "projects/windows-cloud/global/images/family/windows-server-2016-core",
          "sizeGb": "50"
        }
      ]
    },
    "bootstrap": {
      "createInstances": [
        {
          "name": "bootstrap",
          "attachedDisks": ["bootstrap"],
          "metadata": {
            "test_metadata": "this was a test"
          },
          "machineType": "n1-standard-1",
          "startupScript": "shutdown /h"
        }
      ]
    },
    "bootstrap stopped": {
      "timeout": "1h",
      "waitForInstancesStopped": ["bootstrap"]
    }
  },
  "dependencies": {
    "bootstrap": ["create disks"],
    "bootstrap stopped": ["bootstrap"]
  }
}
