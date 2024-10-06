# This script will modify the contents of a Windows boot disk using a rescue Windows instance to modify the contents offline.

instance=$1
zone=$2
project=$3
psscript=$4
labelkey=$5

scriptwaitduration=300 # 5 minutes
maxretries=5
sleepduration=30
machineType=e2-highcpu-4
rescueinstance="rescue-$instance"

echo ""

# 1. Verify input
if [[ $instance == "" ]]; then
    echo instance parameter not set.
    invalid=true
fi
if [[ $zone == "" ]]; then
    echo zone parameter not set.
    invalid=true
fi
if [[ $project == "" ]]; then
    echo project parameter not set.
    invalid=true
fi
if [[ $psscript == "" ]]; then
    echo script parameter not set.
    invalid=true
fi

if [[ $invalid ]]; then
    echo ""
    echo Usage of recover_windows_instance.sh instance zone project script.ps1 label
    echo ""
    echo 1. Required instance: The name of the Windows instance that has the boot disk that needs to be modified.
    echo 2. Required zone: The zone to use for compute resources
    echo 3. Required project: The name of the project the Compute resources exist in.
    echo 4. Required script: The PowerShell script that will be run in the rescue instance to modify the contents the specified instances boot disk.
    echo 5. Optional label: If the instance has this label the script will abort. On completion of the modification this label will be added to the instance.
    echo ""
    echo example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project remediationscript.ps1
    echo example: ./recover_windows_instance.sh instance1 us-central1-c my-gcp-project remediationscript.ps1 remediationprojectlabel
    echo ""
    exit
fi


# 2. Verify instance exists
verifyInstanceOutput=$(gcloud compute instances list --zones="${zone}" --project="${project}" --filter=name=$instance)
if [[ $verifyInstanceOutput == *$instance* ]]; then
    echo Found instance $instance in zone $zone of project $project
else
    echo Unable to find $instance in zone $zone of project $project, aborting.
    exit
fi

# 3. Identify if the instance has the label, if present script exits.
if [[ $labelkey != "" ]]; then
    labelcheck=$(gcloud compute instances describe "${instance}" --zone="${zone}" --project="${project}" --format="get(labels[$labelkey])")
    if [[ labelcheck != "" ]]; then
      echo $labelkey found on $instance in zone $zone of project $project. Exiting.
      exit
    fi
fi

# 4. Obtain the boot disk for the instance 
bootdisk=$(gcloud compute instances describe "${instance}" --zone="${zone}" --project="${project}" --format='get(disks[0].source)')


# 5. Stop the instance if it not already TERMINATED.
vmstatecheck=$(gcloud compute instances describe "${instance}" --zone="${zone}" --project="${project}" --format="value(status)")

for (( i = 0 ; i < "${maxretries}" ; i++ )); do
    if [[ ! "${vmstatecheck}" = "TERMINATED" ]]; then
        gcloud compute instances stop "${instance}" --zone="${zone}" --project="${project}"
        vmstatecheck=$(gcloud compute instances describe "${instance}" --zone="${zone}" --project="${project}" --format="value(status)")
    fi
    if [[ ! "${vmstatecheck}" = "TERMINATED" ]]; then
        echo "VM not TERMINATED. Waiting ${sleepduration} sec."
        if [[ "${i}" -eq "${maxretries}" ]]; then
            echo "VM not TERMINATED after ${maxretries} attempts. Giving up."
            exit
        fi
        sleep ${sleepduration}
    fi
done

echo Detaching bootdisk $bootdisk from $instance in zone $zone of project $project.
# 6. Detaches the boot disk from the instance.
gcloud compute instances detach-disk "${instance}" --disk="${bootdisk}" --zone="${zone}" --project="${project}"

# 7. Creates a rescue Windows instance and mounts theoriginial boot disks as a data driver (D:).
gcloud compute instances create "${rescueinstance}" --image-project=windows-cloud --image-family=windows-2022-core --machine-type=$machineType --zone="${zone}" --project="${project}" --disk=auto-delete=false,name="${bootdisk}" --metadata-from-file=windows-startup-script-ps1="${psscript}"

# 8. Wait for x seconds to boot, run the rescue script, and shutdown.
sleep "${scriptwaitduration}"

# 9. Verify the rescue instance is stopped.
for (( i = 0 ; i < "${maxretries}" ; i++ )); do
  echo "Querying VM status to check for TERMINATED state."
  vmstatecheck=$(gcloud compute instances describe "${rescueinstance}" --zone="${zone}" --project="${project}" --format="value(status)")
  if [[ "${vmstatecheck}" = "TERMINATED" ]]; then
    echo "VM in TERMINATED state."
    break
  else
    echo "State not TERMINATED. Waiting ${sleepduration} sec."
    if [[ "${i}" -eq "${maxretries}" ]]; then
      echo "rescue-$instance not TERMINATED after ${maxretries} attempts. Giving up."
      break
    fi
    sleep "${sleepduration}"
  fi
done

# 10. Detach the boot disk from "${rescueinstance}".
gcloud compute instances detach-disk "${rescueinstance}" --disk="${bootdisk}" --zone="${zone}" --project="${project}"

# 11. Re-attach the boot disk to original instance.
gcloud compute instances attach-disk "${instance}" --disk="${bootdisk}" --boot --zone="${zone}" --project="${project}"

# 12. Deleting rescue instance.
gcloud compute instances delete "${rescueinstance}" --zone="${zone}" --project="${project}"

# 13. If a label has been specified, set that label on the instance.
if [[ $labelkey != "" ]]; then
    gcloud compute instances add-labels "${instance}" --labels=$labelkey=$(date +'%F_%H%M%S%z') --zone="${zone}" --project="${project}"
fi