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
	"bytes"
	"strings"
	"text/template"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	upgradeIntroductionTemplate = "The following resources will be created/accessed during the upgrade. " +
		"Please note the names of the following resources in case you need to manually rollback or cleanup resources.\n" +
		"All resources are in project '{{.project}}', zone '{{.zone}}'.\n" +
		"1. Instance: {{.instanceName}}\n" +
		"2. Disk for install media: {{.installMediaDiskName}}\n" +
		"3. Snapshot for original OS disk: {{.osDiskSnapshotName}}\n" +
		"4. Original OS disk: {{.osDiskName}}\n" +
		"   - Device name of the attachment: {{.osDiskDeviceName}}\n" +
		"   - AutoDelete setting of the attachment: {{.osDiskAutoDelete}}\n" +
		"5. Name of the new OS disk: {{.newOSDiskName}}\n" +
		"6. Name of the machine image: {{.machineImageName}}\n" +
		"7. Original startup script URL: {{.originalStartupScriptURL}}\n" +
		"\n" +
		"If the upgrade succeeds but the cleanup fails, use the following steps to perform a manual cleanup:\n" +
		"1. Delete 'windows-startup-script-url' from the instance's metadata if there isn't an original value. " +
		"If there is an original value, restore it. The original value is backed up as metadata 'windows-startup-script-url-backup'.\n" +
		"2. Detach the install media disk from the instance and delete it.\n" +
		"\n" +
		"If the upgrade fails but you didn't enable automatic rollback, auto-rollback " +
		"failed, or the upgrade succeeded but you need to rollback for another reason, " +
		"use the following steps to perform a manual rollback:\n" +
		"1. Detach the new OS disk from the instance and delete the disk.\n" +
		"2. Attach the original OS disk as a boot disk.\n" +
		"3. Detach the install media disk from the instance and delete the disk.\n" +
		"4. Delete 'windows-startup-script-url' from the instance's metadata if there isn't an original value for the script. " +
		"If there is an original value for the script, restore the value. The original value is backed up as metadata 'windows-startup-script-url-backup'.\n" +
		"\n" +
		"After verifying that the upgrading succeeds and you no longer need to rollback:\n" +
		"1. Delete the original OS disk.\n" +
		"2. Delete the machine image.\n" +
		"3. Delete the snapshot.\n" +
		"\n"
)

var (
	supportedSourceOSVersions    = map[string]string{versionWindows2008r2: versionWindows2012r2}
	supportedTargetOSVersions, _ = param.ReverseMap(supportedSourceOSVersions)
)

// SupportedSourceOSVersions returns supported source versions of upgrading
func SupportedSourceOSVersions() []string {
	return param.GetKeys(supportedSourceOSVersions)
}

// SupportedTargetOSVersions returns supported target versions of upgrading
func SupportedTargetOSVersions() []string {
	return param.GetKeys(supportedTargetOSVersions)
}

func getIntroHelpText(u *upgrader) (string, error) {
	originalStartupScriptURL := "None."
	if u.originalWindowsStartupScriptURL != nil {
		originalStartupScriptURL = *u.originalWindowsStartupScriptURL
	}
	if u.machineImageBackupName == "" {
		u.machineImageBackupName = "Not created. Machine Image backup is disabled."
	}

	t, err := template.New("guide").Option("missingkey=error").Parse(upgradeIntroductionTemplate)
	if err != nil {
		return "", daisy.Errf("Failed to parse upgrade guide.")
	}
	var buf bytes.Buffer
	varMap := map[string]interface{}{
		"project":                  u.instanceProject,
		"zone":                     u.instanceZone,
		"instanceName":             u.instanceName,
		"installMediaDiskName":     u.installMediaDiskName,
		"osDiskSnapshotName":       u.osDiskSnapshotName,
		"osDiskName":               daisyutils.GetResourceRealName(u.osDiskURI),
		"osDiskDeviceName":         u.osDiskDeviceName,
		"osDiskAutoDelete":         u.osDiskAutoDelete,
		"newOSDiskName":            u.newOSDiskName,
		"machineImageName":         u.machineImageBackupName,
		"originalStartupScriptURL": originalStartupScriptURL,
	}
	if err := t.Execute(&buf, varMap); err != nil {
		return "", daisy.Errf("Failed to generate upgrade guide.")
	}
	return string(buf.Bytes()), nil
}

func isNewOSDiskAttached(project, zone, instanceName, newOSDiskName string) bool {
	inst, err := computeClient.GetInstance(project, zone, instanceName)
	if err != nil {
		// failed to fetch info. Can't guarantee new OS disk is attached.
		return false
	}

	// If the "prepare" workflow failed when the original OS disk has been dettached
	// but new OS disk hasn't been attached, we won't find a boot disk from the instance.
	// The instance will either have no disk (while it had only a boot disk originally),
	// or have some data disks (while it had more than one disks originally).
	// Boot disk is always with index=0: https://cloud.google.com/compute/docs/reference/rest/v1/instances/attachDisk
	// "0 is reserved for the boot disk"
	if len(inst.Disks) == 0 || inst.Disks[0].Boot == false {
		// if the instance has no boot disk attached
		return false
	}

	currentBootDiskURL := inst.Disks[0].Source

	// ignore project / zone, only compare real name, because it's guaranteed that
	// old OS disk and new OS disk are in the same project and zone.
	currentBootDiskName := daisyutils.GetResourceRealName(currentBootDiskURL)
	return currentBootDiskName == newOSDiskName
}

func needReboot(err error) bool {
	// windows-2008r2 will emit this error string to the serial port when a
	// restarting is required
	return strings.Contains(err.Error(), "Windows needs to be restarted")
}

func setWorkflowAttributes(w *daisy.Workflow, u *upgrader) {
	daisycommon.SetWorkflowAttributes(w, u.instanceProject, u.instanceZone, u.ScratchBucketGcsPath,
		u.Oauth, u.Timeout, u.Ce, u.GcsLogsDisabled, u.CloudLogsDisabled, u.StdoutLogsDisabled)
}
