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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

// InstanceExportPreparer prepares a Compute Engine instance for export into OVF
// by shutting it down and detaching disks
type InstanceExportPreparer interface {
	Prepare(instance *compute.Instance) error
	TraceLogs() []string
	Cancel(reason string) bool
}

type instanceExportPreparerImpl struct {
	wf                *daisy.Workflow
	params            *OVFExportParams
	workflowGenerator *OVFExportWorkflowGenerator
	instance          *compute.Instance
	workflowPath      string
	serialLogs        []string
}

// NewInstanceExportPreparer creates a new instance export preparer
func NewInstanceExportPreparer(params *OVFExportParams, workflowGenerator *OVFExportWorkflowGenerator,
	workflowPath string) InstanceExportPreparer {
	return &instanceExportPreparerImpl{
		params:            params,
		workflowGenerator: workflowGenerator,
		workflowPath:      workflowPath,
	}
}

func (iep *instanceExportPreparerImpl) Prepare(instance *compute.Instance) error {
	iep.instance = instance
	var err error
	iep.wf, err = runWorkflowWithSteps(context.Background(), "ovf-export-prepare",
		iep.workflowPath, iep.params.Timeout, func(w *daisy.Workflow) error { return iep.populatePrepareSteps(w, instance) },
		map[string]string{}, iep.params)
	if iep.wf.Logger != nil {
		iep.serialLogs = iep.wf.Logger.ReadSerialPortLogs()
	}
	return err
}

func (iep *instanceExportPreparerImpl) populatePrepareSteps(w *daisy.Workflow, instance *compute.Instance) error {
	var previousStepName string
	if isInstanceRunning(iep.instance) {
		previousStepName = "stop-instance"
		iep.workflowGenerator.addStopInstanceStep(w, instance, previousStepName)
	}

	_, err := iep.workflowGenerator.AddDetachDisksSteps(w, instance, previousStepName, "")

	if err != nil {
		return err
	}
	return nil
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
