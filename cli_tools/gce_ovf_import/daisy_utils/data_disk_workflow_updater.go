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
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	"google.golang.org/api/compute/v1"
)

const (
	createInstanceStepName = "create-instance"
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

		setupDataDiskStep := daisy.NewStepDefaultTimeout(setupDataDiskStepName, w)
		setupDataDiskStep.CreateDisksAlpha = &daisy.CreateDisksBeta{
			{
				Disk: computeAlpha.Disk{
					Name:                diskNames[i],
					Type:                "pd-ssd",
					SourceStorageObject: dataDiskFilePath,
				},
				DiskBase: daisy.DiskBase{
					FallbackToPdStandard: true,
					Resource: daisy.Resource{
						ExactName: true,
						NoCleanup: true,
					},
				},
				SizeGb: "10",
			},
		}
		w.Steps[setupDataDiskStepName] = setupDataDiskStep

		w.Dependencies[createInstanceStepName] = append(
			w.Dependencies[createInstanceStepName], setupDataDiskStepName)
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
