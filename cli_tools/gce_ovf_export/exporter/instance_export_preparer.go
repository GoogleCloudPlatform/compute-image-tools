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
	"fmt"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type instanceExportPreparerImpl struct {
	worker           daisyutils.DaisyWorker
	instance         *compute.Instance
	logger           logging.Logger
	wfPreRunCallback wfCallback
}

// NewInstanceExportPreparer creates a new instance export preparer
func NewInstanceExportPreparer(logger logging.Logger) ovfexportdomain.InstanceExportPreparer {
	return &instanceExportPreparerImpl{logger: logger}
}

func (iep *instanceExportPreparerImpl) Prepare(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) error {
	wfName := "ovf-export-prepare"
	iep.instance = instance
	workflowProvider := func() (*daisy.Workflow, error) {
		wf, err := generateWorkflowWithSteps(wfName, "30m", func(w *daisy.Workflow) error { return iep.populatePrepareSteps(w, instance, params) })
		if err != nil {
			return nil, err
		}
		if iep.wfPreRunCallback != nil {
			iep.wfPreRunCallback(wf)
		}
		return wf, nil
	}

	iep.worker = daisyutils.NewDaisyWorker(workflowProvider, params.EnvironmentSettings(wfName), iep.logger)
	return iep.worker.Run(map[string]string{})
}

func (iep *instanceExportPreparerImpl) populatePrepareSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) error {
	var previousStepName string
	if isInstanceRunning(iep.instance) {
		previousStepName = "stop-instance"
		iep.addStopInstanceStep(w, instance, previousStepName, params)
	}
	if err := iep.addDetachDisksSteps(w, instance, previousStepName, params); err != nil {
		return err
	}
	return nil
}

// addDetachDisksSteps adds Daisy steps to OVF export workflow to detach instance disks.
func (iep *instanceExportPreparerImpl) addDetachDisksSteps(w *daisy.Workflow,
	instance *compute.Instance, previousStepName string, params *ovfexportdomain.OVFExportArgs) error {
	if instance == nil || len(instance.Disks) == 0 {
		return daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks
	for i, attachedDisk := range attachedDisks {
		detachDiskStepName := fmt.Sprintf("detach-disk-%v-%v", i, attachedDisk.DeviceName)
		detachDiskStep := daisy.NewStepDefaultTimeout(detachDiskStepName, w)
		detachDiskStep.DetachDisks = &daisy.DetachDisks{
			&daisy.DetachDisk{
				Instance:   getInstancePath(instance, params.Project),
				DeviceName: daisyutils.GetDeviceURI(params.Project, params.Zone, attachedDisk.DeviceName),
			},
		}

		w.Steps[detachDiskStepName] = detachDiskStep
		if previousStepName != "" {
			w.Dependencies[detachDiskStepName] = append(w.Dependencies[detachDiskStepName], previousStepName)
		}
	}
	return nil
}

func (iep *instanceExportPreparerImpl) Cancel(reason string) bool {
	if iep.worker == nil {
		return false
	}
	iep.worker.Cancel(reason)
	return true
}

// addStopInstanceStep adds a StopInstance step to a workflow
func (iep *instanceExportPreparerImpl) addStopInstanceStep(w *daisy.Workflow,
	instance *compute.Instance, stopInstanceStepName string, params *ovfexportdomain.OVFExportArgs) {
	stopInstanceStep := daisy.NewStepDefaultTimeout(stopInstanceStepName, w)
	stopInstanceStep.StopInstances = &daisy.StopInstances{
		Instances: []string{getInstancePath(instance, params.Project)},
	}
	w.Steps[stopInstanceStepName] = stopInstanceStep
}
