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
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	daisyovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/daisy_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type populateStepsFunc func(*daisy.Workflow) error

func (oe *OVFExporter) prepare() (*daisy.Workflow, error) {
	if oe.prepareFn != nil {
		return oe.prepareFn()
	}
	return oe.runWorkflowWithSteps("ovf-export-prepare", oe.params.Timeout, oe.populatePrepareSteps)
}

func (oe *OVFExporter) populatePrepareSteps(w *daisy.Workflow) error {
	isInstanceRunning := isInstanceRunning(oe.instance)

	workflowGenerator := &daisyovfutils.OVFExportWorkflowGenerator{
		Instance:               oe.instance,
		Project:                *oe.params.Project,
		Zone:                   oe.params.Zone,
		OvfGcsDirectoryPath:    oe.params.DestinationURI,
		ExportedDiskFileFormat: oe.params.DiskExportFormat,
		Network:                oe.params.Network,
		Subnet:                 oe.params.Subnet,
		InstancePath:           oe.instancePath,
		IsInstanceRunning:      isInstanceRunning,
	}

	var previousStepName string
	if isInstanceRunning {
		previousStepName = "stop-instance"
		workflowGenerator.AddStopInstanceStep(w, previousStepName)
	}

	_, err := workflowGenerator.AddDetachDisksSteps(w, previousStepName, "")

	if err != nil {
		return err
	}
	return nil
}

func (oe *OVFExporter) exportDisks() (*daisy.Workflow, error) {
	if oe.prepareFn != nil {
		return oe.prepareFn()
	}
	return oe.runWorkflowWithSteps("ovf-export-disk-run", oe.params.Timeout, oe.populateExportDisksSteps)
}

func (oe *OVFExporter) populateExportDisksSteps(w *daisy.Workflow) error {
	isInstanceRunning := isInstanceRunning(oe.instance)

	workflowGenerator := &daisyovfutils.OVFExportWorkflowGenerator{
		Instance:               oe.instance,
		Project:                *oe.params.Project,
		Zone:                   oe.params.Zone,
		OvfGcsDirectoryPath:    oe.params.DestinationURI,
		ExportedDiskFileFormat: oe.params.DiskExportFormat,
		Network:                oe.params.Network,
		Subnet:                 oe.params.Subnet,
		InstancePath:           oe.instancePath,
		IsInstanceRunning:      isInstanceRunning,
	}

	_, _, err := workflowGenerator.AddExportDisksSteps(w, []string{}, "")

	if err != nil {
		return err
	}
	return nil
}

func (oe *OVFExporter) runWorkflowWithSteps(workflowName string, timeout string, populateStepsFunc populateStepsFunc) (*daisy.Workflow, error) {
	w, err := oe.generateWorkflowWithSteps(workflowName, timeout, populateStepsFunc)
	if err != nil {
		return w, err
	}

	setWorkflowAttributes(w, oe)
	err = daisyutils.RunWorkflowWithCancelSignal(oe.ctx, w)
	return w, err
}

func setWorkflowAttributes(w *daisy.Workflow, oe *OVFExporter) {
	daisycommon.SetWorkflowAttributes(w, *oe.params.Project, oe.zone, oe.params.ScratchBucketGcsPath,
		oe.params.Oauth, oe.params.Timeout, oe.params.Ce, oe.params.GcsLogsDisabled, oe.params.CloudLogsDisabled, oe.params.StdoutLogsDisabled)
}

func (oe *OVFExporter) generateWorkflowWithSteps(workflowName string, timeout string, populateStepsFunc populateStepsFunc) (*daisy.Workflow, error) {
	varMap := oe.buildDaisyVars()

	w, err := daisycommon.ParseWorkflow(oe.workflowPath, varMap, *oe.params.Project,
		oe.zone, oe.params.ScratchBucketGcsPath, oe.params.Oauth, oe.params.Timeout, oe.params.Ce,
		oe.params.GcsLogsDisabled, oe.params.CloudLogsDisabled, oe.params.StdoutLogsDisabled)
	if err != nil {
		return w, err
	}
	w.Name = workflowName
	w.DefaultTimeout = timeout
	w.ForceCleanupOnError = true
	w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)
	err = populateStepsFunc(w)
	return w, err
}
