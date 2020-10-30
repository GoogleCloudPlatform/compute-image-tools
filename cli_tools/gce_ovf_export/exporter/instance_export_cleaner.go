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

package ovfexporter

import (
	"context"
	"fmt"
	"strings"

	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

type instanceExportCleanerImpl struct {
	wf           *daisy.Workflow
	instance     *compute.Instance
	workflowPath string
	serialLogs   []string
}

// NewInstanceExportCleaner creates a new instance export cleaner
func NewInstanceExportCleaner(workflowPath string) ovfexportdomain.InstanceExportCleaner {
	return &instanceExportCleanerImpl{
		workflowPath: workflowPath,
	}
}

func (iec *instanceExportCleanerImpl) Clean(instance *compute.Instance, params *ovfexportdomain.OVFExportParams) error {
	iec.instance = instance
	var err error
	iec.wf, err = runWorkflowWithSteps(context.Background(), "ovf-export-clean",
		iec.workflowPath, params.Timeout.String(), func(w *daisy.Workflow) error { return iec.populateCleanupSteps(w, instance, params) },
		map[string]string{}, params)
	if iec.wf.Logger != nil {
		iec.serialLogs = iec.wf.Logger.ReadSerialPortLogs()
	}
	return err
}

func (iec *instanceExportCleanerImpl) populateCleanupSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportParams) error {
	var nextStepName string
	var err error
	if isInstanceRunning(instance) {
		nextStepName = "start-instance"
	}
	_, err = iec.addAttachDisksSteps(w, instance, params, nextStepName)
	if err != nil {
		return err
	}
	if isInstanceRunning(instance) {
		iec.addStartInstanceStep(w, instance, params, nextStepName)
	}
	return nil
}

// addAttachDisksSteps adds Daisy steps to OVF export workflow to attach disks back to the instance.
func (iec *instanceExportCleanerImpl) addAttachDisksSteps(w *daisy.Workflow,
	instance *compute.Instance, params *ovfexportdomain.OVFExportParams, nextStepName string) ([]string, error) {
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
				Instance: getInstancePath(instance, *params.Project),
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

// addStartInstanceStep adds a StartInstance step to a workflow
func (iec *instanceExportCleanerImpl) addStartInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, params *ovfexportdomain.OVFExportParams, startInstanceStepName string) {
	startInstanceStep := daisy.NewStepDefaultTimeout(startInstanceStepName, w)
	startInstanceStep.StartInstances = &daisy.StartInstances{Instances: []string{getInstancePath(instance, *params.Project)}}
	w.Steps[startInstanceStepName] = startInstanceStep
}

func (iec *instanceExportCleanerImpl) TraceLogs() []string {
	return iec.serialLogs
}

func (iec *instanceExportCleanerImpl) Cancel(reason string) bool {
	if iec.wf == nil {
		return false
	}
	iec.wf.CancelWithReason(reason)
	return true
}
