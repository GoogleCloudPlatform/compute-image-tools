{
  "steps": {
    "create-disks": {
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
    "bootstrap-stopped": {
      "timeout": "1h",
      "waitForInstancesSignal": [
        {"name": "bootstrap", "serialOutput": {"port": 1, "successMatch": "complete", "failureMatch": "fail"}}
      ]
    }
  },
  "dependencies": {
    "bootstrap": ["create-disks"],
    "bootstrap-stopped": ["bootstrap"]
  }
}
