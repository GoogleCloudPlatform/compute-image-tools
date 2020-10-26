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

package ovfexporter

import (
	"fmt"
	"strconv"
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

// OVFExportWorkflowGenerator generates OVF export workflow
type OVFExportWorkflowGenerator struct {
	Project                string
	Zone                   string
	OvfGcsDirectoryPath    string
	ExportedDiskFileFormat string
	Network                string
	Subnet                 string
	InstancePath           string
}

// AddDetachDisksSteps adds Daisy steps to OVF export workflow to detach instance disks.
func (g *OVFExportWorkflowGenerator) AddDetachDisksSteps(w *daisy.Workflow, instance *compute.Instance, previousStepName, nextStepName string) ([]string, error) {
	if instance == nil || len(instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks

	var stepNames []string

	for i, attachedDisk := range attachedDisks {
		detachDiskStepName := fmt.Sprintf("detach-disk-%v-%v", i, attachedDisk.DeviceName)
		stepNames = append(stepNames, detachDiskStepName)
		detachDiskStep := daisy.NewStepDefaultTimeout(detachDiskStepName, w)
		detachDiskStep.DetachDisks = &daisy.DetachDisks{
			&daisy.DetachDisk{
				Instance:   g.getInstancePath(instance),
				DeviceName: daisyutils.GetDeviceURI(g.Project, g.Zone, attachedDisk.DeviceName),
			},
		}

		w.Steps[detachDiskStepName] = detachDiskStep
		if previousStepName != "" {
			w.Dependencies[detachDiskStepName] = append(w.Dependencies[detachDiskStepName], previousStepName)
		}
		if nextStepName != "" {
			w.Dependencies[nextStepName] = append(w.Dependencies[nextStepName], detachDiskStepName)
		}
	}
	return stepNames, nil
}

func (g *OVFExportWorkflowGenerator) getInstancePath(instance *compute.Instance) string {
	return fmt.Sprintf("projects/%s/zones/%s/instances/%s", g.Project, g.Zone, strings.ToLower(instance.Name))
}

// addExportDisksSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (g *OVFExportWorkflowGenerator) addExportDisksSteps(w *daisy.Workflow, instance *compute.Instance, previousStepNames []string, nextStepName string) ([]*ExportedDisk, error) {
	if instance == nil || len(instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks
	var exportedDisks []*ExportedDisk

	for i, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		exportedDiskGCSPath := g.OvfGcsDirectoryPath + attachedDisk.DeviceName + "." + g.ExportedDiskFileFormat
		exportedDisks = append(exportedDisks, &ExportedDisk{attachedDisk: attachedDisk, gcsPath: exportedDiskGCSPath})

		exportDiskStepName := fmt.Sprintf(
			"export-disk-%v-%v",
			i,
			stringutils.Substring(attachedDisk.DeviceName,
				0,
				63-len("detach-disk-")-len("disk--buffer-12345")-len(strconv.Itoa(i))-2),
		)
		exportDiskStepName = strings.Trim(exportDiskStepName, "-")
		exportDiskStep := daisy.NewStepDefaultTimeout(exportDiskStepName, w)
		exportDiskStep.IncludeWorkflow = &daisy.IncludeWorkflow{
			Path: "../export/disk_export_ext.wf.json",
			Vars: map[string]string{
				"source_disk":                diskPath,
				"destination":                exportedDiskGCSPath,
				"format":                     g.ExportedDiskFileFormat,
				"export_instance_disk_image": "projects/compute-image-tools/global/images/family/debian-9-worker",
				"export_instance_disk_size":  "200",
				"export_instance_disk_type":  "pd-ssd",
				"export_network":             g.Network,
				"export_subnet":              g.Subnet,
				"export_disk_ext.sh":         "../export/export_disk_ext.sh",
				"disk_resizing_mon.sh":       "../export/disk_resizing_mon.sh",
			},
		}

		w.Steps[exportDiskStepName] = exportDiskStep
		if len(previousStepNames) > 0 {
			for _, previousStepName := range previousStepNames {
				w.Dependencies[exportDiskStepName] = append(w.Dependencies[exportDiskStepName], previousStepName)
			}
		}
		if nextStepName != "" {
			w.Dependencies[nextStepName] = append(w.Dependencies[nextStepName], exportDiskStepName)
		}
	}
	return exportedDisks, nil
}

// addAttachDisksSteps adds Daisy steps to OVF export workflow to attach disks back to the instance.
func (g *OVFExportWorkflowGenerator) addAttachDisksSteps(w *daisy.Workflow,
	instance *compute.Instance, nextStepName string) ([]string, error) {
	if instance == nil || len(instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks

	var stepNames []string

	for _, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		attachDiskStepName := fmt.Sprintf("attach-disk-%v", attachedDisk.DeviceName)
		stepNames = append(stepNames, attachDiskStepName)
		attachDiskStep := daisy.NewStepDefaultTimeout(attachDiskStepName, w)
		attachDiskStep.AttachDisks = &daisy.AttachDisks{
			{
				Instance: g.getInstancePath(instance),
				AttachedDisk: compute.AttachedDisk{
					Mode:       attachedDisk.Mode,
					Source:     diskPath,
					Boot:       attachedDisk.Boot,
					DeviceName: attachedDisk.DeviceName,
				},
			},
		}

		w.Steps[attachDiskStepName] = attachDiskStep
		if nextStepName != "" {
			w.Dependencies[nextStepName] = append(w.Dependencies[nextStepName], attachDiskStepName)
		}
	}
	return stepNames, nil
}

// addStopInstanceStep adds a StopInstance step to a workflow
func (g *OVFExportWorkflowGenerator) addStopInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, stopInstanceStepName string) {
	stopInstanceStep := daisy.NewStepDefaultTimeout(stopInstanceStepName, w)
	stopInstanceStep.StopInstances = &daisy.StopInstances{
		Instances: []string{g.getInstancePath(instance)},
	}
	w.Steps[stopInstanceStepName] = stopInstanceStep
}

// addStartInstanceStep adds a StartInstance step to a workflow
func (g *OVFExportWorkflowGenerator) addStartInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, startInstanceStepName string) {
	startInstanceStep := daisy.NewStepDefaultTimeout(startInstanceStepName, w)
	startInstanceStep.StartInstances = &daisy.StartInstances{Instances: []string{g.getInstancePath(instance)}}
	w.Steps[startInstanceStepName] = startInstanceStep
}
