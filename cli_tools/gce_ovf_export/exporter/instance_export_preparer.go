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

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

type instanceExportPreparerImpl struct {
	wf           *daisy.Workflow
	instance     *compute.Instance
	workflowPath string
	serialLogs   []string
}

// NewInstanceExportPreparer creates a new instance export preparer
func NewInstanceExportPreparer(workflowPath string) ovfexportdomain.InstanceExportPreparer {
	return &instanceExportPreparerImpl{
		workflowPath: workflowPath,
	}
}

func (iep *instanceExportPreparerImpl) Prepare(instance *compute.Instance, params *ovfexportdomain.OVFExportParams) error {
	iep.instance = instance
	var err error
	iep.wf, err = runWorkflowWithSteps(context.Background(), "ovf-export-prepare",
		iep.workflowPath, params.Timeout.String(), func(w *daisy.Workflow) error { return iep.populatePrepareSteps(w, instance, params) },
		map[string]string{}, params)
	if iep.wf.Logger != nil {
		iep.serialLogs = iep.wf.Logger.ReadSerialPortLogs()
	}
	return err
}

func (iep *instanceExportPreparerImpl) populatePrepareSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportParams) error {
	var previousStepName string
	if isInstanceRunning(iep.instance) {
		previousStepName = "stop-instance"
		iep.addStopInstanceStep(w, instance, previousStepName, params)
	}

	_, err := iep.addDetachDisksSteps(w, instance, previousStepName, params)

	if err != nil {
		return err
	}
	return nil
}

// addDetachDisksSteps adds Daisy steps to OVF export workflow to detach instance disks.
func (iep *instanceExportPreparerImpl) addDetachDisksSteps(w *daisy.Workflow, instance *compute.Instance, previousStepName string, params *ovfexportdomain.OVFExportParams) ([]string, error) {
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
				Instance:   getInstancePath(instance, *params.Project),
				DeviceName: daisyutils.GetDeviceURI(*params.Project, params.Zone, attachedDisk.DeviceName),
			},
		}

		w.Steps[detachDiskStepName] = detachDiskStep
		if previousStepName != "" {
			w.Dependencies[detachDiskStepName] = append(w.Dependencies[detachDiskStepName], previousStepName)
		}
	}
	return stepNames, nil
}

func (iep *instanceExportPreparerImpl) TraceLogs() []string {
	return iep.serialLogs
}

func (iep *instanceExportPreparerImpl) Cancel(reason string) bool {
	if iep.wf == nil {
		return false
	}
	iep.wf.CancelWithReason(reason)
	return true
}

// addStopInstanceStep adds a StopInstance step to a workflow
func (iep *instanceExportPreparerImpl) addStopInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, stopInstanceStepName string, params *ovfexportdomain.OVFExportParams) {
	stopInstanceStep := daisy.NewStepDefaultTimeout(stopInstanceStepName, w)
	stopInstanceStep.StopInstances = &daisy.StopInstances{
		Instances: []string{getInstancePath(instance, *params.Project)},
	}
	w.Steps[stopInstanceStepName] = stopInstanceStep
}
