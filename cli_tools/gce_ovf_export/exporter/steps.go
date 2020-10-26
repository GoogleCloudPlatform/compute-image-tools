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
	"sync"

	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

type populateStepsFunc func(*daisy.Workflow) error

func (oe *OVFExporter) prepare(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		return oe.instanceExportPreparer.Prepare(instance)
	}, oe.instanceExportPreparer.Cancel, oe.instanceExportPreparer.TraceLogs)
}

func (oe *OVFExporter) exportDisks(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		var err error
		oe.exportedDisks, err = oe.instanceDisksExporter.Export(instance)
		return err
	}, oe.instanceDisksExporter.Cancel, oe.instanceDisksExporter.TraceLogs)
}

func (oe *OVFExporter) inspectBootDisk(ctx context.Context) error {
	bootDisk := getBootDisk(oe.exportedDisks)
	if bootDisk == nil {
		return nil
	}
	return oe.runStep(ctx, func() error {
		var err error
		oe.bootDiskInspectionResults, err = oe.inspector.Inspect(
			daisyutils.GetDiskURI(*oe.params.Project, oe.params.Zone, bootDisk.disk.Name), true)
		if err != nil {
			oe.Logger.Log(fmt.Sprintf("WARNING: Could not detect operating system on the boot disk: %v", err))
		}
		oe.Logger.Log(fmt.Sprintf("Disk inspection results: %v", oe.bootDiskInspectionResults))
		// don't return error if inspection fails, just log it, since it's not a show-stopper.
		return nil
	}, oe.inspector.Cancel, oe.inspector.TraceLogs)
}

func (oe *OVFExporter) generateDescriptor(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		if bucketName, gcsDirectoryPath, err := storageutils.GetGCSObjectPathElements(oe.params.DestinationURI); err != nil {
			return err
		} else {
			return oe.ovfDescriptorGenerator.GenerateAndWriteOVFDescriptor(instance, oe.exportedDisks, bucketName, gcsDirectoryPath, &oe.bootDiskInspectionResults)
		}
	}, oe.ovfDescriptorGenerator.Cancel, func() []string { return nil })
}

func (oe *OVFExporter) generateManifest(ctx context.Context) error {
	return oe.runStep(ctx, func() error {
		return oe.manifestFileGenerator.GenerateAndWriteToGCS(oe.params.DestinationURI, oe.params.InstanceName)
	}, oe.manifestFileGenerator.Cancel, func() []string { return nil })
}

func (oe *OVFExporter) cleanup(ctx context.Context, instance *compute.Instance) error {
	_, err := runWorkflowWithSteps(context.Background(), "ovf-export-cleanup",
		oe.workflowPath, oe.params.Timeout,
		func(w *daisy.Workflow) error { return populateCleanupSteps(w, oe.workflowGenerator, instance) },
		map[string]string{}, oe.params)
	if err != nil {
		return err
	}
	if oe.storageClient != nil {
		err := oe.storageClient.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func populateCleanupSteps(w *daisy.Workflow, workflowGenerator *OVFExportWorkflowGenerator, instance *compute.Instance) error {
	var nextStepName string
	var err error
	if isInstanceRunning(instance) {
		nextStepName = "start-instance"
	}
	_, err = workflowGenerator.addAttachDisksSteps(w, instance, nextStepName)
	if err != nil {
		return err
	}
	if isInstanceRunning(instance) {
		workflowGenerator.addStartInstanceStep(w, instance, nextStepName)
	}
	return nil
}

func runWorkflowWithSteps(ctx context.Context, workflowName, workflowPath, timeout string,
	populateStepsFunc populateStepsFunc, varMap map[string]string, params *OVFExportParams) (*daisy.Workflow, error) {

	w, err := generateWorkflowWithSteps(workflowName, workflowPath, timeout, populateStepsFunc, varMap, params)
	if err != nil {
		return w, err
	}

	daisycommon.SetWorkflowAttributes(w, params.DaisyAttrs())
	err = daisyutils.RunWorkflowWithCancelSignal(ctx, w)
	return w, err
}

func generateWorkflowWithSteps(workflowName, workflowPath, timeout string, populateStepsFunc populateStepsFunc, varMap map[string]string, params *OVFExportParams) (*daisy.Workflow, error) {
	w, err := daisycommon.ParseWorkflow(workflowPath, varMap, *params.Project,
		params.Zone, params.ScratchBucketGcsPath, params.Oauth, params.Timeout, params.Ce,
		params.GcsLogsDisabled, params.CloudLogsDisabled, params.StdoutLogsDisabled)
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

//TODO: consolidate with gce_vm_image_import.runStep()
func (oe *OVFExporter) runStep(ctx context.Context, step func() error, cancel func(string) bool, getTraceLogs func() []string) (err error) {
	e := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		//this select checks if context expired prior to runStep being called
		//if not, step is run
		select {
		case <-ctx.Done():
			e <- oe.getCtxError(ctx)
		default:
			stepErr := step()
			wg.Done()
			e <- stepErr
		}
	}()

	// this select waits for either context expiration or step to finish (with either an error or success)
	select {
	case <-ctx.Done():
		if cancel("timed-out") {
			//Only return timeout error if step was able to cancel on time-out.
			//Otherwise, step has finished and import succeeded even though it timed out
			err = oe.getCtxError(ctx)
		}
		wg.Wait()
	case stepErr := <-e:
		err = stepErr
	}
	if getTraceLogs != nil {
		stepTraceLogs := getTraceLogs()
		if stepTraceLogs != nil && len(stepTraceLogs) > 0 {
			oe.traceLogs = append(oe.traceLogs, stepTraceLogs...)
		}
	}
	return err
}

//TODO: consolidate with gce_vm_image_import.getCtxError()
func (oe *OVFExporter) getCtxError(ctx context.Context) (err error) {
	if ctxErr := ctx.Err(); ctxErr == context.DeadlineExceeded {
		err = daisy.Errf("Import did not complete within the specified timeout of %s", oe.timeout)
	} else {
		err = ctxErr
	}
	return err
}

func (oe *OVFExporter) buildDaisyVars() map[string]string {
	varMap := map[string]string{}
	//TODO: nework/subnet are needed for export workflow. Can it be set in a different way instead having it as a param?

	//if oe.params.IsInstanceExport() {
	//	// instance import specific vars
	//	varMap["instance_name"] = oe.instancePath
	//} else {
	//	// machine image import specific vars
	//	varMap["machine_image_name"] = strings.ToLower(oe.params.MachineImageName)
	//}
	//
	//if oe.params.Subnet != "" {
	//	varMap["subnet"] = param.GetRegionalResourcePath(oe.region, "subnetworks", oe.params.Subnet)
	//	// When subnet is set, we need to grant a value to network to avoid fallback to default
	//	if oe.params.Network == "" {
	//		varMap["network"] = ""
	//	}
	//}
	//if oe.params.Network != "" {
	//	varMap["network"] = param.GetGlobalResourcePath("networks", oe.params.Network)
	//}
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

func getBootDisk(exportedDisks []*ExportedDisk) *ExportedDisk {
	for _, exportedDisk := range exportedDisks {
		if exportedDisk.attachedDisk.Boot {
			return exportedDisk
		}
	}
	return nil
}
