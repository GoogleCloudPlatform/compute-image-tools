# GCE Recovery Scripts

The following are scripts that utilize gcloud to automate the recovery of an instance that is not able to successfully boot.

## How to use the recovery scripts using Cloud Shell

1. Open a new Cloud Shell terminal. https://cloud.google.com/shell/docs/using-cloud-shell#start_a_new_session
2. git clone https://github.com/GoogleCloudPlatform/compute-image-tools.git
3. cd ~/compute-image-tools/gce_recovery_scripts
4. run ./recover_windows_instance.sh with the needed parameters.

## Windows instance recovery

`recover_windows_instance.sh` automates the creation of a rescue Windows instance that is used to modify the contents of the specified instances boot disk using a startup script.

### Usage

    The `recover_windows_instance.sh` supports the following parameters:
    
    1. Required `instance`: The name of the Windows instance that has the boot disk that needs to be modified.
    2. Required `zone`: The zone to use for compute resources
    3. Required `project`: The name of the project the Compute resources exist in.
    4. Required `script`: The PowerShell script that will be run in the rescue instance to modify the contents the specified instances boot disk.
    5. Optional `label`: If the instance has this label the script will abort. On completion of the modification this label will be added to the instance.
    
    example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project remediationscript.ps1
    example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project remediationscript.ps1 remediationprojectlabel

### recover_windows_instance.sh does the following:

1. Validates the required input parameters of the script are present.
2. Verifies the instance exists.
   - Exits if the instance is not present.
3. Checks that the instance does not have the optionally specified label.
   - Exits if the instance has the specified label.
4. Obtain the boot disk for the instance.
5. Stop the instance if it not already TERMINATED.
6. Detaches the boot disk from the instance.
7. Creates a rescue Windows instance, mounting the originial boot disks as a data disk (D:), and specifying the provided script as the startup script (windows-startup-script-ps1).
8. Waiting for 300 second to boot, run the rescue script, and shutdown.
9. Verify the rescue Windows instance is stopped.
10. Detaches the data disk from the rescue Windows instance".
11. Re-attach the boot disk to original instance
12. Deletes the rescue Windows instance
13. If a label has been specified, set that label on the instance.


## Windows GVNIC Driver Recovery

`reinstall-gvnic-gq.ps1` does the following:

1. Downloads the known good version of the gVNIC Windows driver from gs://gce-windows-drivers-public/release/gvnic-gq
2. Removes all versions of the gVNIC Windows driver from the Windows installed on the D:\ drive.
3. Installs the known good gVNIC 1.0.x version of the driver to the Windows installed on the D:\ drive.
4. Shuts downs the rescue instance.

`reinstall-gvnic-gq.ps1` can be used as the `recover_windows_instance.sh` remediation script.

example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project reinstall-gvnic-gq.ps1
example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project reinstall-gvnic-gq.ps1 gvnic-remediation