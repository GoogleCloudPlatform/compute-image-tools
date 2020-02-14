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
//  limitations under the License

package daisyovfutils

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

const (
	createInstanceStepName = "create-Instance"
	importerDiskSize       = "10"
)

type OVFExportWorkflowGenerator struct {
	Instance               *compute.Instance
	Project                string
	Zone                   string
	OvfGcsDirectoryPath    string
	ExportedDiskFileFormat string
	Network                string
	Subnet                 string
	InstancePath           string
	PreviousStepName       string
}

// AddDiskExportSteps adds Daisy steps to OVF export workflow to export disks.
func (g *OVFExportWorkflowGenerator) AddDiskExportSteps(w *daisy.Workflow) ([]string, error) {
	if g.Instance == nil || len(g.Instance.Disks) == 0 {
		return nil, daisy.Errf("No disks found in the Instance to export")
	}
	disks := g.Instance.Disks

	var exportedDisksGCSPaths []string
	w.Sources["export_disk_ext.sh"] = "../export/export_disk_ext.sh"
	w.Sources["disk_resizing_mon.sh"] = "../export/disk_resizing_mon.sh"

	for i, disk := range disks {
		dataDiskIndex := i + 1


		diskPath := disk.Source[strings.Index(disk.Source, "projects/"):]
		exportedDiskGCSPath := g.OvfGcsDirectoryPath + disk.DeviceName
		exportedDisksGCSPaths = append(exportedDisksGCSPaths, exportedDiskGCSPath)

		detachDiskStepName := fmt.Sprintf("detach-disk-%v", dataDiskIndex)
		detachDiskStep := daisy.NewStepDefaultTimeout(detachDiskStepName, w)
		detachDiskStep.DetachDisks = &daisy.DetachDisks{
			{
				Instance:   g.InstancePath,
				DeviceName: diskPath,
			},
		}

		exportDiskStepName := fmt.Sprintf("export-disk-%v", dataDiskIndex)
		exportDiskStep := daisy.NewStepDefaultTimeout(exportDiskStepName, w)
		exportDiskStep.IncludeWorkflow = &daisy.IncludeWorkflow{
			Path: "../export/disk_export_ext.wf.json",
			Vars: map[string]string{
				"source_disk":                disk.DeviceName,
				"destination":                exportedDiskGCSPath,
				"format":                     g.ExportedDiskFileFormat,
				"export_instance_disk_image": "projects/compute-image-tools/global/images/family/debian-9-worker",
				"export_instance_disk_size":  "200",
				"export_instance_disk_type":  "pd-ssd",
				"export_network":             g.Network,
				"export_subnet":              g.Subnet,
			},
		}

		w.Steps[detachDiskStepName] = detachDiskStep
		w.Steps[exportDiskStepName] = exportDiskStep
		w.Dependencies[detachDiskStepName] = append(w.Dependencies[detachDiskStepName], g.PreviousStepName)
		w.Dependencies[exportDiskStepName] = append(w.Dependencies[exportDiskStepName], detachDiskStepName)
	}
	return exportedDisksGCSPaths, nil
}
