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
	"strconv"
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

// OVFExportWorkflowGenerator generates OVF export workflow
type OVFExportWorkflowGenerator struct {
	Instance               *compute.Instance
	Project                string
	Zone                   string
	OvfGcsDirectoryPath    string
	ExportedDiskFileFormat string
	Network                string
	Subnet                 string
	InstancePath           string
	IsInstanceRunning      bool
}

// AddDiskExportSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (g *OVFExportWorkflowGenerator) AddDiskExportSteps(w *daisy.Workflow, previousStepName, nextStepName string) ([]string, error) {
	if g.Instance == nil || len(g.Instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := g.Instance.Disks

	var exportedDisksGCSPaths []string

	for i, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		exportedDiskGCSPath := g.OvfGcsDirectoryPath + attachedDisk.DeviceName + "." + g.ExportedDiskFileFormat
		exportedDisksGCSPaths = append(exportedDisksGCSPaths, exportedDiskGCSPath)

		detachDiskStepName := fmt.Sprintf("detach-disk-%v-%v", i, attachedDisk.DeviceName)
		detachDiskStep := daisy.NewStepDefaultTimeout(detachDiskStepName, w)
		detachDiskStep.DetachDisks = &daisy.DetachDisks{
			&daisy.DetachDisk{
				Instance:   g.InstancePath,
				DeviceName: daisyutils.GetDeviceURI(g.Project, g.Zone, attachedDisk.DeviceName),
			},
		}

		exportDiskStepName := fmt.Sprintf("export-disk-%v-%v", i, stringutils.Substring(attachedDisk.DeviceName, 0, 63-len("detach-disk-")-len("disk--buffer-12345")-len(strconv.Itoa(i))-2))
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

		attachDiskStepName := fmt.Sprintf("attach-disk-%v", attachedDisk.DeviceName)
		attachDiskStep := daisy.NewStepDefaultTimeout(attachDiskStepName, w)
		attachDiskStep.AttachDisks = &daisy.AttachDisks{
			{
				Instance: g.InstancePath,
				AttachedDisk: compute.AttachedDisk{
					Mode:       attachedDisk.Mode,
					Source:     diskPath,
					Boot:       attachedDisk.Boot,
					DeviceName: attachedDisk.DeviceName,
				},
			},
		}

		w.Steps[detachDiskStepName] = detachDiskStep
		w.Steps[exportDiskStepName] = exportDiskStep
		w.Steps[attachDiskStepName] = attachDiskStep
		if previousStepName != "" {
			w.Dependencies[detachDiskStepName] = append(w.Dependencies[detachDiskStepName], previousStepName)
		}
		w.Dependencies[exportDiskStepName] = append(w.Dependencies[exportDiskStepName], detachDiskStepName)
		w.Dependencies[attachDiskStepName] = append(w.Dependencies[attachDiskStepName], exportDiskStepName)
		if nextStepName != "" {
			w.Dependencies[nextStepName] = append(w.Dependencies[nextStepName], attachDiskStepName)
		}
	}
	return exportedDisksGCSPaths, nil
}

// AddDetachDisksSteps adds Daisy steps to OVF export workflow to detach instance disks.
func (g *OVFExportWorkflowGenerator) AddDetachDisksSteps(w *daisy.Workflow, previousStepName, nextStepName string) ([]string, error) {
	if g.Instance == nil || len(g.Instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := g.Instance.Disks

	var stepNames []string

	for i, attachedDisk := range attachedDisks {
		detachDiskStepName := fmt.Sprintf("detach-disk-%v-%v", i, attachedDisk.DeviceName)
		stepNames = append(stepNames, detachDiskStepName)
		detachDiskStep := daisy.NewStepDefaultTimeout(detachDiskStepName, w)
		detachDiskStep.DetachDisks = &daisy.DetachDisks{
			&daisy.DetachDisk{
				Instance:   g.InstancePath,
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

// AddExportDisksSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (g *OVFExportWorkflowGenerator) AddExportDisksSteps(w *daisy.Workflow, previousStepNames []string, nextStepName string) ([]string, []string, error) {
	if g.Instance == nil || len(g.Instance.Disks) == 0 {
		return nil, nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := g.Instance.Disks

	var exportedDisksGCSPaths []string
	var stepNames []string

	for i, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		exportedDiskGCSPath := g.OvfGcsDirectoryPath + attachedDisk.DeviceName + "." + g.ExportedDiskFileFormat
		exportedDisksGCSPaths = append(exportedDisksGCSPaths, exportedDiskGCSPath)

		exportDiskStepName := fmt.Sprintf("export-disk-%v-%v", i, stringutils.Substring(attachedDisk.DeviceName, 0, 63-len("detach-disk-")-len("disk--buffer-12345")-len(strconv.Itoa(i))-2))
		exportDiskStepName = strings.Trim(exportDiskStepName, "-")
		stepNames = append(stepNames, exportDiskStepName)
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
	return exportedDisksGCSPaths, stepNames, nil
}

// AddDiskExportSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (g *OVFExportWorkflowGenerator) AddAttachDisksSteps(w *daisy.Workflow, previousStepNames []string, nextStepName string) ([]string, error) {
	if g.Instance == nil || len(g.Instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := g.Instance.Disks

	var stepNames []string

	for _, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		attachDiskStepName := fmt.Sprintf("attach-disk-%v", attachedDisk.DeviceName)
		stepNames = append(stepNames, attachDiskStepName)
		attachDiskStep := daisy.NewStepDefaultTimeout(attachDiskStepName, w)
		attachDiskStep.AttachDisks = &daisy.AttachDisks{
			{
				Instance: g.InstancePath,
				AttachedDisk: compute.AttachedDisk{
					Mode:       attachedDisk.Mode,
					Source:     diskPath,
					Boot:       attachedDisk.Boot,
					DeviceName: attachedDisk.DeviceName,
				},
			},
		}

		w.Steps[attachDiskStepName] = attachDiskStep
		if len(previousStepNames) > 0 {
			for _, previousStepName := range previousStepNames {
				w.Dependencies[attachDiskStepName] = append(w.Dependencies[attachDiskStepName], previousStepName)
			}
		}
		if nextStepName != "" {
			w.Dependencies[nextStepName] = append(w.Dependencies[nextStepName], attachDiskStepName)
		}
	}
	return stepNames, nil
}

// AddStopInstanceStep adds a StopInstance step to a workflow
func (g *OVFExportWorkflowGenerator) AddStopInstanceStep(w *daisy.Workflow, stopInstanceStepName string) {
	stopInstanceStep := daisy.NewStepDefaultTimeout(stopInstanceStepName, w)
	stopInstanceStep.StopInstances = &daisy.StopInstances{
		Instances: []string{g.InstancePath},
	}
	w.Steps[stopInstanceStepName] = stopInstanceStep
}

// AddStartInstanceStep adds a StartInstance step to a workflow
func (g *OVFExportWorkflowGenerator) AddStartInstanceStep(w *daisy.Workflow, startInstanceStepName string, previousStepNames []string) {
	startInstanceStep := daisy.NewStepDefaultTimeout(startInstanceStepName, w)
	startInstanceStep.StartInstances = &daisy.StartInstances{Instances: []string{g.InstancePath}}
	w.Steps[startInstanceStepName] = startInstanceStep

	if len(previousStepNames) > 0 {
		for _, previousStepName := range previousStepNames {
			w.Dependencies[previousStepName] = append(w.Dependencies[previousStepName], startInstanceStepName)
		}
	}
}
