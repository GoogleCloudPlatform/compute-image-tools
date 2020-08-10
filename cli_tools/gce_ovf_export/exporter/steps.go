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
	"strings"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	compute "google.golang.org/api/compute/v1"
)

type populateStepsFunc func(*daisy.Workflow) error

func (oe *OVFExporter) prepare() (*daisy.Workflow, error) {
	if oe.prepareFn != nil {
		return oe.prepareFn()
	}
	return oe.runWorkflowWithSteps("ovf-export-prepare", oe.params.Timeout, oe.populatePrepareSteps)
}

func (oe *OVFExporter) exportDisks() (*daisy.Workflow, error) {
	if oe.exportDisksFn != nil {
		return oe.exportDisksFn()
	}
	return oe.runWorkflowWithSteps("ovf-export-disk-export", oe.params.Timeout, oe.populateExportDisksSteps)
}

func (oe *OVFExporter) generateDescriptor() error {
	if oe.generateDescriptorFn != nil {
		return oe.generateDescriptorFn()
	}

	// populate exported disks with compute.Disk and storage object attributes
	// that are needed for descriptor generation
	for _, exportedDisk := range oe.exportedDisks {

		// populate compute.Disk for each exported disk
		if disk, err := oe.computeClient.GetDisk(*oe.params.Project, oe.params.Zone, daisyutils.GetResourceID(exportedDisk.attachedDisk.Source)); err == nil {
			exportedDisk.disk = disk
		} else {
			return err
		}

		// populate storage object attributes for each exported disk file
		bucketName, objectPath, err := storageutils.SplitGCSPath(exportedDisk.gcsPath)
		if err != nil {
			return err
		}
		exportedDisk.gcsFileAttrs, err = oe.storageClient.GetObjectAttrs(bucketName, objectPath)
		if err != nil {
			return err
		}
	}
	if err := oe.ovfDescriptorGenerator.GenerateAndWriteOVFDescriptor(oe.instance, oe.exportedDisks, oe.bucketName, oe.gcsDirectoryPath); err != nil {
		return err
	}
	return nil
}

func (oe *OVFExporter) cleanup() (*daisy.Workflow, error) {
	if oe.cleanupFn != nil {
		return oe.cleanupFn()
	}
	wf, err := oe.runWorkflowWithSteps("ovf-export-cleanup", oe.params.Timeout, oe.populateCleanupSteps)
	if err != nil {
		return wf, err
	}
	if oe.storageClient != nil {
		err := oe.storageClient.Close()
		if err != nil {
			return wf, err
		}
	}
	return wf, nil
}

func (oe *OVFExporter) populatePrepareSteps(w *daisy.Workflow) error {
	var previousStepName string
	if isInstanceRunning(oe.instance) {
		previousStepName = "stop-instance"
		oe.workflowGenerator.AddStopInstanceStep(w, previousStepName)
	}

	_, err := oe.workflowGenerator.AddDetachDisksSteps(w, previousStepName, "")

	if err != nil {
		return err
	}
	return nil
}

func (oe *OVFExporter) populateExportDisksSteps(w *daisy.Workflow) error {
	var err error
	oe.exportedDisks, err = oe.workflowGenerator.AddExportDisksSteps(w, []string{}, "")
	if err != nil {
		return err
	}

	return nil
}

func (oe *OVFExporter) populateCleanupSteps(w *daisy.Workflow) error {
	var nextStepName string
	var err error
	if isInstanceRunning(oe.instance) {
		nextStepName = "start-instance"
	}
	_, err = oe.workflowGenerator.AddAttachDisksSteps(w, nextStepName)
	if err != nil {
		return err
	}
	if isInstanceRunning(oe.instance) {
		oe.workflowGenerator.AddStartInstanceStep(w, nextStepName)
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
	daisycommon.SetWorkflowAttributes(w, daisycommon.WorkflowAttributes{
		Project:           *oe.params.Project,
		Zone:              oe.params.Zone,
		GCSPath:           oe.params.ScratchBucketGcsPath,
		OAuth:             oe.params.Oauth,
		Timeout:           oe.params.Timeout,
		ComputeEndpoint:   oe.params.Ce,
		DisableGCSLogs:    oe.params.GcsLogsDisabled,
		DisableCloudLogs:  oe.params.CloudLogsDisabled,
		DisableStdoutLogs: oe.params.StdoutLogsDisabled,
	})
}

func (oe *OVFExporter) generateWorkflowWithSteps(workflowName string, timeout string, populateStepsFunc populateStepsFunc) (*daisy.Workflow, error) {
	varMap := oe.buildDaisyVars()

	w, err := daisycommon.ParseWorkflow(oe.workflowPath, varMap, *oe.params.Project,
		oe.params.Zone, oe.params.ScratchBucketGcsPath, oe.params.Oauth, oe.params.Timeout, oe.params.Ce,
		oe.params.GcsLogsDisabled, oe.params.CloudLogsDisabled, oe.params.StdoutLogsDisabled)
	if err != nil {
		return w, err
	}
	w.Name = workflowName
	w.DefaultTimeout = timeout
	w.ForceCleanupOnError = true
	w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)

	if err = populateStepsFunc(w); err != nil {
		return w, err
	}

	//TODO: this cannot be done with included workflows. Refactor included workflows by in-lining them
	//oe.labelResources(w)
	//daisyutils.UpdateAllInstanceNoExternalIP(w, oe.params.NoExternalIP)

	return w, err
}

func (oe *OVFExporter) buildDaisyVars() map[string]string {
	varMap := map[string]string{}
	if oe.params.IsInstanceExport() {
		// instance import specific vars
		varMap["instance_name"] = oe.instancePath
	} else {
		// machine image import specific vars
		varMap["machine_image_name"] = strings.ToLower(oe.params.MachineImageName)
	}

	if oe.params.Subnet != "" {
		varMap["subnet"] = param.GetRegionalResourcePath(oe.region, "subnetworks", oe.params.Subnet)
		// When subnet is set, we need to grant a value to network to avoid fallback to default
		if oe.params.Network == "" {
			varMap["network"] = ""
		}
	}
	if oe.params.Network != "" {
		varMap["network"] = param.GetGlobalResourcePath("networks", oe.params.Network)
	}
	return varMap
}

func (oe *OVFExporter) labelResources(w *daisy.Workflow) {
	rl := &daisyutils.ResourceLabeler{
		BuildID:         oe.BuildID,
		BuildIDLabelKey: "gce-ovf-export-build-id",
		InstanceLabelKeyRetriever: func(instanceName string) string {
			return "gce-ovf-export-tmp"
		},
		DiskLabelKeyRetriever: func(disk *daisy.Disk) string {
			return "gce-ovf-export-tmp"
		},
		ImageLabelKeyRetriever: func(imageName string) string {
			return "gce-ovf-export-tmp"
		}}
	rl.LabelResources(w)
}

func isInstanceRunning(instance *compute.Instance) bool {
	return !(instance == nil || instance.Status == "STOPPED" || instance.Status == "STOPPING" ||
		instance.Status == "SUSPENDED" || instance.Status == "SUSPENDING")
}
