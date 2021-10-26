//  Copyright 2020 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package upgrader

import (
	"fmt"
	"os"

	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

var (
	upgradeSteps      = map[string]func(*upgrader, *daisy.Workflow) error{versionWindows2008r2: populateUpgradeStepsFrom2008r2To2012r2}
	retryUpgradeSteps = map[string]func(*upgrader, *daisy.Workflow) error{versionWindows2008r2: populateRetryUpgradeStepsFrom2008r2To2012r2}
)

func (u *upgrader) prepare() (*daisy.Workflow, error) {
	if u.prepareFn != nil {
		return u.prepareFn()
	}

	return u.runWorkflowWithSteps("windows-upgrade-preparation", u.Timeout, populatePrepareSteps)
}

func populatePrepareSteps(u *upgrader, w *daisy.Workflow) error {
	currentExecutablePath := os.Args[0]
	w.Sources = map[string]string{"upgrade_script.ps1": path.ToWorkingDir(upgradeScriptName[u.SourceOS], currentExecutablePath)}

	stepStopInstance, err := daisyutils.NewStep(w, "stop-instance")
	if err != nil {
		return err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.instanceURI},
	}
	prevStep := stepStopInstance

	if u.CreateMachineBackup {
		stepBackupMachineImage, err := daisyutils.NewStep(w, "backup-machine-image", stepStopInstance)
		if err != nil {
			return err
		}
		stepBackupMachineImage.CreateMachineImages = &daisy.CreateMachineImages{
			&daisy.MachineImage{
				MachineImage: computeBeta.MachineImage{
					Name:           u.machineImageBackupName,
					SourceInstance: u.instanceURI,
				},
				Resource: daisy.Resource{
					ExactName: true,
					NoCleanup: true,
				},
			},
		}
		prevStep = stepBackupMachineImage
	}

	stepBackupOSDiskSnapshot, err := daisyutils.NewStep(w, "backup-os-disk-snapshot", prevStep)
	if err != nil {
		return err
	}
	stepBackupOSDiskSnapshot.CreateSnapshots = &daisy.CreateSnapshots{
		&daisy.Snapshot{
			Snapshot: compute.Snapshot{
				Name:       u.osDiskSnapshotName,
				SourceDisk: u.osDiskURI,
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepCreateNewOSDisk, err := daisyutils.NewStep(w, "create-new-os-disk", stepBackupOSDiskSnapshot)
	if err != nil {
		return err
	}
	stepCreateNewOSDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name:           u.newOSDiskName,
				Zone:           u.instanceZone,
				Type:           u.osDiskType,
				SourceSnapshot: u.osDiskSnapshotName,
				Licenses:       []string{licenseToAdd[u.SourceOS]},
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}

	stepDetachOldOSDisk, err := daisyutils.NewStep(w, "detach-old-os-disk", stepCreateNewOSDisk)
	if err != nil {
		return err
	}
	stepDetachOldOSDisk.DetachDisks = &daisy.DetachDisks{
		&daisy.DetachDisk{
			Instance:   u.instanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.instanceProject, u.instanceZone, u.osDiskDeviceName),
		},
	}

	stepAttachNewOSDisk, err := daisyutils.NewStep(w, "attach-new-os-disk", stepDetachOldOSDisk)
	if err != nil {
		return err
	}
	stepAttachNewOSDisk.AttachDisks = &daisy.AttachDisks{
		&daisy.AttachDisk{
			Instance: u.instanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.newOSDiskName,
				DeviceName: u.osDiskDeviceName,
				AutoDelete: u.osDiskAutoDelete,
				Boot:       true,
			},
		},
	}

	stepCreateInstallDisk, err := daisyutils.NewStep(w, "create-install-disk", stepAttachNewOSDisk)
	if err != nil {
		return err
	}
	stepCreateInstallDisk.CreateDisks = &daisy.CreateDisks{
		&daisy.Disk{
			Disk: compute.Disk{
				Name:        u.installMediaDiskName,
				Zone:        u.instanceZone,
				Type:        "pd-ssd",
				SourceImage: "projects/compute-image-tools/global/images/family/windows-install-media",
			},
			Resource: daisy.Resource{
				ExactName: true,
				NoCleanup: true,
			},
		},
	}
	if u.UseStagingInstallMedia {
		(*stepCreateInstallDisk.CreateDisks)[0].SourceImage = "projects/bct-prod-images/global/images/family/windows-install-media"
	}

	stepAttachInstallDisk, err := daisyutils.NewStep(w, "attach-install-disk", stepCreateInstallDisk)
	if err != nil {
		return err
	}
	stepAttachInstallDisk.AttachDisks = &daisy.AttachDisks{
		&daisy.AttachDisk{
			Instance: u.instanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.installMediaDiskName,
				AutoDelete: true,
			},
		},
	}
	prevStep = stepAttachInstallDisk

	// If there isn't an original url, just skip the backup step.
	if u.originalWindowsStartupScriptURL != nil {
		fmt.Printf("\nDetected an existing metadata for key '%v', value='%v'. Will backup to '%v'.\n\n", metadataKeyWindowsStartupScriptURL,
			*u.originalWindowsStartupScriptURL, metadataKeyWindowsStartupScriptURLBackup)

		stepBackupScript, err := daisyutils.NewStep(w, "backup-script", stepAttachInstallDisk)
		if err != nil {
			return err
		}
		stepBackupScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
			&daisy.UpdateInstanceMetadata{
				Instance: u.instanceURI,
				Metadata: map[string]string{metadataKeyWindowsStartupScriptURLBackup: *u.originalWindowsStartupScriptURL},
			},
		}
		prevStep = stepBackupScript
	}

	stepSetScript, err := daisyutils.NewStep(w, "set-script", prevStep)
	if err != nil {
		return err
	}
	stepSetScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		&daisy.UpdateInstanceMetadata{
			Instance: u.instanceURI,
			Metadata: map[string]string{metadataKeyWindowsStartupScriptURL: "${SOURCESPATH}/upgrade_script.ps1"},
		},
	}
	return nil
}

func (u *upgrader) upgrade() (*daisy.Workflow, error) {
	if u.upgradeFn != nil {
		return u.upgradeFn()
	}

	return u.runWorkflowWithSteps("upgrade", u.Timeout, upgradeSteps[u.SourceOS])
}

func populateUpgradeStepsFrom2008r2To2012r2(u *upgrader, w *daisy.Workflow) error {
	cleanupWorkflow, err := u.generateWorkflowWithSteps("cleanup", "10m", populateCleanupSteps)
	if err != nil {
		return nil
	}

	w.Steps = map[string]*daisy.Step{
		"start-instance": {
			StartInstances: &daisy.StartInstances{
				Instances: []string{u.instanceURI},
			},
		},
		"wait-for-boot": {
			Timeout: "15m",
			WaitForInstancesSignal: &daisy.WaitForInstancesSignal{
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "GCEMetadataScripts: Beginning upgrade startup script.",
					},
				},
			},
		},
		"wait-for-upgrade": {
			WaitForAnyInstancesSignal: &daisy.WaitForAnyInstancesSignal{
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "windows_upgrade_current_version=6.3",
						FailureMatch: []string{"UpgradeFailed:"},
						StatusMatch:  "GCEMetadataScripts:",
					},
				},
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port: 3,
						// These errors were thrown from setup.exe.
						FailureMatch: []string{"Windows needs to be restarted", "CheckDiskSpaceRequirements not satisfied"},
						// This is the prefix of error log emitted from install media. Catch it and write to daisy log for debugging.
						StatusMatch: "$WINDOWS.~BT setuperr$",
					},
				},
			},
		},
		"cleanup-temp-resources": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: cleanupWorkflow,
			},
		},
	}
	w.Dependencies = map[string][]string{
		"wait-for-boot":          {"start-instance"},
		"wait-for-upgrade":       {"start-instance"},
		"cleanup-temp-resources": {"wait-for-upgrade"},
	}
	return nil
}

func (u *upgrader) retryUpgrade() (*daisy.Workflow, error) {
	if u.retryUpgradeFn != nil {
		return u.retryUpgradeFn()
	}

	return u.runWorkflowWithSteps("retry-upgrade", u.Timeout, retryUpgradeSteps[u.SourceOS])
}

func populateRetryUpgradeStepsFrom2008r2To2012r2(u *upgrader, w *daisy.Workflow) error {
	cleanupWorkflow, err := u.generateWorkflowWithSteps("cleanup", "10m", populateCleanupSteps)
	if err != nil {
		return nil
	}

	w.Steps = map[string]*daisy.Step{
		"wait-for-boot": {
			Timeout: "15m",
			WaitForInstancesSignal: &daisy.WaitForInstancesSignal{
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "GCEMetadataScripts: Beginning upgrade startup script.",
					},
				},
			},
		},
		"wait-for-upgrade": {
			WaitForAnyInstancesSignal: &daisy.WaitForAnyInstancesSignal{
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port:         1,
						SuccessMatch: "windows_upgrade_current_version=6.3",
						FailureMatch: []string{"UpgradeFailed:"},
						StatusMatch:  "GCEMetadataScripts:",
					},
				},
				{
					Name: u.instanceURI,
					SerialOutput: &daisy.SerialOutput{
						Port: 3,
						// These errors were thrown from setup.exe.
						FailureMatch: []string{"Windows needs to be restarted", "CheckDiskSpaceRequirements not satisfied"},
					},
				},
			},
		},
		"cleanup-temp-resources": {
			IncludeWorkflow: &daisy.IncludeWorkflow{
				Workflow: cleanupWorkflow,
			},
		},
	}
	w.Dependencies = map[string][]string{
		"cleanup-temp-resources": {"wait-for-upgrade"},
	}
	return nil
}

func (u *upgrader) reboot() (*daisy.Workflow, error) {
	if u.rebootFn != nil {
		return u.rebootFn()
	}

	return u.runWorkflowWithSteps("reboot", "15m", populateRebootSteps)
}

func populateRebootSteps(u *upgrader, w *daisy.Workflow) error {
	w.Steps = map[string]*daisy.Step{
		"stop-instance": {
			StopInstances: &daisy.StopInstances{
				Instances: []string{u.instanceURI},
			},
		},
		"start-instance": {
			StartInstances: &daisy.StartInstances{
				Instances: []string{u.instanceURI},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"start-instance": {"stop-instance"},
	}
	return nil
}

func (u *upgrader) cleanup() (*daisy.Workflow, error) {
	if u.cleanupFn != nil {
		return u.cleanupFn()
	}

	return u.runWorkflowWithSteps("cleanup", "20m", populateCleanupSteps)
}

func populateCleanupSteps(u *upgrader, w *daisy.Workflow) error {
	w.Steps = map[string]*daisy.Step{
		"restore-script": {
			UpdateInstancesMetadata: &daisy.UpdateInstancesMetadata{
				{
					Instance: u.instanceURI,
					Metadata: map[string]string{
						metadataKeyWindowsStartupScriptURL:       u.getOriginalStartupScriptURL(),
						metadataKeyWindowsStartupScriptURLBackup: "",
					},
				},
			},
		},
		"detach-install-media-disk": {
			DetachDisks: &daisy.DetachDisks{
				{
					Instance:   u.instanceURI,
					DeviceName: daisyutils.GetDeviceURI(u.instanceProject, u.instanceZone, u.installMediaDiskName),
				},
			},
		},
		"delete-install-media-disk": {
			DeleteResources: &daisy.DeleteResources{
				Disks: []string{
					daisyutils.GetDiskURI(u.instanceProject, u.instanceZone, u.installMediaDiskName),
				},
			},
		},
		// TODO: use a flag to determine whether to stop the instance. b/156668741
		"stop-instance": {
			StopInstances: &daisy.StopInstances{
				Instances: []string{u.instanceURI},
			},
		},
	}
	w.Dependencies = map[string][]string{
		"delete-install-media-disk": {"detach-install-media-disk"},
	}
	return nil
}

func (u *upgrader) rollback() (*daisy.Workflow, error) {
	if u.rollbackFn != nil {
		return u.rollbackFn()
	}

	return u.runWorkflowWithSteps("rollback", u.Timeout, populateRollbackSteps)
}

func populateRollbackSteps(u *upgrader, w *daisy.Workflow) error {
	stepStopInstance, err := daisyutils.NewStep(w, "stop-instance")
	if err != nil {
		return err
	}
	stepStopInstance.StopInstances = &daisy.StopInstances{
		Instances: []string{u.instanceURI},
	}

	stepDetachNewOSDisk, err := daisyutils.NewStep(w, "detach-new-os-disk", stepStopInstance)
	if err != nil {
		return err
	}
	stepDetachNewOSDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.instanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.instanceProject, u.instanceZone, u.osDiskDeviceName),
		},
	}

	stepAttachOldOSDisk, err := daisyutils.NewStep(w, "attach-old-os-disk", stepDetachNewOSDisk)
	if err != nil {
		return err
	}
	stepAttachOldOSDisk.AttachDisks = &daisy.AttachDisks{
		{
			Instance: u.instanceURI,
			AttachedDisk: compute.AttachedDisk{
				Source:     u.osDiskURI,
				DeviceName: u.osDiskDeviceName,
				AutoDelete: u.osDiskAutoDelete,
				Boot:       true,
			},
		},
	}

	stepDetachInstallMediaDisk, err := daisyutils.NewStep(w, "detach-install-media-disk", stepAttachOldOSDisk)
	if err != nil {
		return err
	}
	stepDetachInstallMediaDisk.DetachDisks = &daisy.DetachDisks{
		{
			Instance:   u.instanceURI,
			DeviceName: daisyutils.GetDeviceURI(u.instanceProject, u.instanceZone, u.installMediaDiskName),
		},
	}

	stepRestoreScript, err := daisyutils.NewStep(w, "restore-script", stepDetachInstallMediaDisk)
	if err != nil {
		return err
	}
	stepRestoreScript.UpdateInstancesMetadata = &daisy.UpdateInstancesMetadata{
		{
			Instance: u.instanceURI,
			Metadata: map[string]string{
				metadataKeyWindowsStartupScriptURL:       u.getOriginalStartupScriptURL(),
				metadataKeyWindowsStartupScriptURLBackup: "",
			},
		},
	}

	// TODO: use a flag to determine whether to start the instance. b/156668741

	stepDeleteNewOSDiskAndInstallMediaDisk, err := daisyutils.NewStep(w, "delete-new-os-disk-and-install-media-disk", stepRestoreScript)
	if err != nil {
		return err
	}
	stepDeleteNewOSDiskAndInstallMediaDisk.DeleteResources = &daisy.DeleteResources{
		Disks: []string{
			daisyutils.GetDiskURI(u.instanceProject, u.instanceZone, u.newOSDiskName),
			daisyutils.GetDiskURI(u.instanceProject, u.instanceZone, u.installMediaDiskName),
		},
	}
	return nil
}

func (u *upgrader) getOriginalStartupScriptURL() string {
	originalStartupScriptURL := ""
	if u.originalWindowsStartupScriptURL != nil {
		originalStartupScriptURL = *u.originalWindowsStartupScriptURL
	}
	return originalStartupScriptURL
}

func (u *upgrader) runWorkflowWithSteps(workflowName string, timeout string, populateStepsFunc func(*upgrader, *daisy.Workflow) error) (*daisy.Workflow, error) {

	var wf *daisy.Workflow

	workflowProvider := func() (*daisy.Workflow, error) {
		var err error
		wf, err = u.generateWorkflowWithSteps(workflowName, timeout, populateStepsFunc)
		return wf, err
	}

	env := daisyutils.EnvironmentSettings{
		Project:           u.instanceProject,
		Zone:              u.instanceZone,
		GCSPath:           u.ScratchBucketGcsPath,
		OAuth:             u.Oauth,
		Timeout:           u.Timeout,
		ComputeEndpoint:   u.Ce,
		DisableGCSLogs:    u.GcsLogsDisabled,
		DisableCloudLogs:  u.CloudLogsDisabled,
		DisableStdoutLogs: u.StdoutLogsDisabled,
		ExecutionID:       u.executionID,
		Tool: daisyutils.Tool{
			HumanReadableName: "windows upgrade",
			ResourceLabelName: "windows-upgrade",
		},
	}

	err := daisyutils.NewDaisyWorker(workflowProvider, env, u.logger).Run(map[string]string{})
	return wf, err
}

func (u *upgrader) generateWorkflowWithSteps(workflowName string, timeout string, populateStepsFunc func(*upgrader, *daisy.Workflow) error) (*daisy.Workflow, error) {
	w := daisy.New()
	w.Name = workflowName
	w.DefaultTimeout = timeout
	err := populateStepsFunc(u, w)
	return w, err
}
