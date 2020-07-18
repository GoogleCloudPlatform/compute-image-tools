//  Copyright 2019 Google Inc. All Rights Reserved.
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
//  limitations under the License

package daisyovfutils

import (
	"fmt"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

const (
	createInstanceStepName = "create-instance"
	gceMinimumDiskSizeGB   = "10"
)

// AddDiskImportSteps adds Daisy steps to OVF import workflow to import disks
// defined in dataDiskInfos.
func AddDiskImportSteps(w *daisy.Workflow, dataDiskInfos []ovfutils.DiskInfo) {
	if dataDiskInfos == nil || len(dataDiskInfos) == 0 {
		return
	}

	var diskNames []string
	w.Sources["import_image_data.sh"] = "../image_import/import_image.sh"

	for i, dataDiskInfo := range dataDiskInfos {
		dataDiskIndex := i + 1
		dataDiskFilePath := dataDiskInfo.FilePath
		diskNames = append(diskNames, generateDataDiskName(w.Vars["instance_name"].Value, dataDiskIndex))

		setupDataDiskStepName := fmt.Sprintf("setup-data-disk-%v", dataDiskIndex)
		diskImporterDiskName := fmt.Sprintf("disk-importer-%v-%v", dataDiskIndex, w.ID())
		scratchDiskDiskName := fmt.Sprintf("disk-importer-scratch-%v-%v", dataDiskIndex, w.ID())

		setupDataDiskStep := daisy.NewStepDefaultTimeout(setupDataDiskStepName, w)
		setupDataDiskStep.CreateDisks = &daisy.CreateDisks{
			{
				Disk: compute.Disk{
					Name:        diskImporterDiskName,
					SourceImage: "projects/compute-image-tools/global/images/family/debian-9-worker",
					Type:        "pd-ssd",
				},
				SizeGb:               gceMinimumDiskSizeGB,
				FallbackToPdStandard: true,
			},
			{
				Disk: compute.Disk{
					Name: diskNames[i],
					Type: "pd-ssd",
				},
				SizeGb:               gceMinimumDiskSizeGB,
				FallbackToPdStandard: true,
				Resource: daisy.Resource{
					ExactName: true,
					NoCleanup: true,
				},
			},
			{
				Disk: compute.Disk{
					Name: scratchDiskDiskName,
					Type: "pd-ssd",
				},
				SizeGb:               gceMinimumDiskSizeGB,
				FallbackToPdStandard: true,
				Resource: daisy.Resource{
					ExactName: true,
				},
			},
		}
		w.Steps[setupDataDiskStepName] = setupDataDiskStep

		createDiskImporterInstanceStepName := fmt.Sprintf("create-data-disk-import-instance-%v", dataDiskIndex)
		createDiskImporterInstanceStep := daisy.NewStepDefaultTimeout(createDiskImporterInstanceStepName, w)

		sTrue := "true"
		diskSizeGB := gceMinimumDiskSizeGB
		dataDiskImporterInstanceName := fmt.Sprintf("data-disk-importer-%v", dataDiskIndex)
		createDiskImporterInstanceStep.CreateInstances = &daisy.CreateInstances{
			Instances: []*daisy.Instance{{
				Instance: compute.Instance{
					Name: dataDiskImporterInstanceName,
					Disks: []*compute.AttachedDisk{
						{Source: diskImporterDiskName},
						{Source: scratchDiskDiskName},
						{Source: diskNames[i]}},
					MachineType: "n1-standard-4",
					Metadata: &compute.Metadata{
						Items: []*compute.MetadataItems{
							{Key: "block-project-ssh-keys", Value: &sTrue},
							{Key: "disk_name", Value: &diskNames[i]},
							{Key: "scratch_disk_name", Value: &scratchDiskDiskName},
							{Key: "source_disk_file", Value: &dataDiskFilePath},
							{Key: "scratch_disk_size_gb", Value: &diskSizeGB},
							{Key: "inflated_disk_size_gb", Value: &diskSizeGB},
						},
					},
					NetworkInterfaces: []*compute.NetworkInterface{
						{
							Network:    w.Vars["network"].Value,
							Subnetwork: w.Vars["subnet"].Value,
						},
					},
				},
				InstanceBase: daisy.InstanceBase{
					Scopes: []string{
						"https://www.googleapis.com/auth/devstorage.read_write",
						"https://www.googleapis.com/auth/compute",
					},
					StartupScript: "import_image_data.sh",
				},
			},
			},
		}
		w.Steps[createDiskImporterInstanceStepName] = createDiskImporterInstanceStep

		waitForDataDiskImportInstanceSignalStepName := fmt.Sprintf("wait-for-data-disk-%v-signal", dataDiskIndex)
		waitForDataDiskImportInstanceSignalStep := daisy.NewStepDefaultTimeout(waitForDataDiskImportInstanceSignalStepName, w)
		waitForDataDiskImportInstanceSignalStep.WaitForInstancesSignal = &daisy.WaitForInstancesSignal{
			{
				Name: dataDiskImporterInstanceName,
				SerialOutput: &daisy.SerialOutput{
					Port:         1,
					SuccessMatch: "ImportSuccess:",
					FailureMatch: []string{"ImportFailed:", "WARNING Failed to download metadata script"},
					StatusMatch:  "Import:",
				},
			},
		}
		w.Steps[waitForDataDiskImportInstanceSignalStepName] = waitForDataDiskImportInstanceSignalStep

		deleteDataDiskImportInstanceSignalStepName := fmt.Sprintf("delete-data-disk-%v-import-instance", dataDiskIndex)
		deleteDataDiskImportInstanceSignalStep := daisy.NewStepDefaultTimeout(deleteDataDiskImportInstanceSignalStepName, w)
		deleteDataDiskImportInstanceSignalStep.DeleteResources = &daisy.DeleteResources{
			Instances: []string{dataDiskImporterInstanceName},
		}
		w.Steps[deleteDataDiskImportInstanceSignalStepName] = deleteDataDiskImportInstanceSignalStep

		w.Dependencies[createDiskImporterInstanceStepName] = []string{setupDataDiskStepName}
		w.Dependencies[waitForDataDiskImportInstanceSignalStepName] = []string{createDiskImporterInstanceStepName}
		w.Dependencies[deleteDataDiskImportInstanceSignalStepName] = []string{waitForDataDiskImportInstanceSignalStepName}

		w.Dependencies[createInstanceStepName] = append(
			w.Dependencies[createInstanceStepName], deleteDataDiskImportInstanceSignalStepName)
	}

	// attach newly created disks to the instance
	for _, diskName := range diskNames {
		(*w.Steps[createInstanceStepName].CreateInstances).Instances[0].Disks =
			append(
				(*w.Steps[createInstanceStepName].CreateInstances).Instances[0].Disks,
				&compute.AttachedDisk{Source: diskName, AutoDelete: true})
	}
}

func generateDataDiskName(instanceName string, dataDiskIndex int) string {
	diskSuffix := fmt.Sprintf("-%v", dataDiskIndex)
	if len(instanceName)+len(diskSuffix) > 63 {
		instanceNameRunes := []rune(instanceName)
		instanceName = string(instanceNameRunes[0 : 63-len(diskSuffix)])
	}
	return fmt.Sprintf("%v%v", instanceName, diskSuffix)
}
