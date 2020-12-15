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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

type populateStepsFunc func(*daisy.Workflow) error

func (oe *OVFExporter) prepare(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		oe.Logger.User(fmt.Sprintf("Stopping '%v' instance and detaching the disks.", instance.Name))
		return oe.instanceExportPreparer.Prepare(instance, oe.params)
	}, oe.instanceExportPreparer.Cancel, oe.instanceExportPreparer.TraceLogs)
}

func (oe *OVFExporter) exportDisks(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		oe.Logger.User("Exporting the disks.")
		var err error
		oe.exportedDisks, err = oe.instanceDisksExporter.Export(instance, oe.params)
		return err
	}, oe.instanceDisksExporter.Cancel, oe.instanceDisksExporter.TraceLogs)
}

func (oe *OVFExporter) inspectBootDisk(ctx context.Context) error {
	return oe.runStep(ctx, func() error {
		oe.Logger.User("Inspecting the boot disk.")
		bootDisk := getBootDisk(oe.exportedDisks)
		if bootDisk == nil {
			return nil
		}
		var err error
		oe.bootDiskInspectionResults, err = oe.inspector.Inspect(
			daisyutils.GetDiskURI(oe.params.Project, oe.params.Zone, bootDisk.Disk.Name), true)
		if err != nil {
			oe.Logger.User(fmt.Sprintf("WARNING: Could not detect operating system on the boot disk: %v", err))
		}
		oe.Logger.User(fmt.Sprintf("Disk inspection results: %v", oe.bootDiskInspectionResults))
		// don't return error if inspection fails, just log it, since it's not a show-stopper.
		return nil
	}, oe.inspector.Cancel, oe.inspector.TraceLogs)
}

func (oe *OVFExporter) generateDescriptor(ctx context.Context, instance *compute.Instance) error {
	return oe.runStep(ctx, func() error {
		oe.Logger.User("Generating OVF descriptor.")
		bucketName, gcsDirectoryPath, err := storageutils.GetGCSObjectPathElements(oe.params.DestinationURI)
		if err != nil {
			return err
		}
		return oe.ovfDescriptorGenerator.GenerateAndWriteOVFDescriptor(instance, oe.exportedDisks, bucketName, gcsDirectoryPath, oe.bootDiskInspectionResults)
	}, oe.ovfDescriptorGenerator.Cancel, func() []string { return nil })
}

func (oe *OVFExporter) generateManifest(ctx context.Context) error {
	return oe.runStep(ctx, func() error {
		oe.Logger.User("Generating manifest.")
		return oe.manifestFileGenerator.GenerateAndWriteToGCS(oe.params.DestinationURI, oe.params.InstanceName)
	}, oe.manifestFileGenerator.Cancel, func() []string { return nil })
}

func (oe *OVFExporter) cleanup(instance *compute.Instance, exportError error) error {
	// cleanup shouldn't react to time out as it's necessary to perform this step.
	// Otherwise, instance being exported would be left shut down and disks detached.
	oe.Logger.User("Cleaning up.")
	if err := oe.instanceExportCleaner.Clean(instance, oe.params); err != nil {
		return err
	}
	if exportError == nil {
		oe.Logger.User("OVF export finished successfully.")
	}
	if oe.storageClient != nil {
		err := oe.storageClient.Close()
		if err != nil {
			return err
		}
	}
	oe.appendTraceLogs(oe.instanceExportCleaner.TraceLogs())
	return nil
}

func generateWorkflowWithSteps(workflowName, timeout string, populateStepsFunc populateStepsFunc,
	params *ovfexportdomain.OVFExportArgs) (*daisy.Workflow, error) {

	w := daisy.New()
	w.Name = workflowName
	w.DefaultTimeout = timeout
	w.ForceCleanupOnError = true
	w.SetLogProcessHook(daisyutils.RemovePrivacyLogTag)

	if err := populateStepsFunc(w); err != nil {
		return w, err
	}

	//This only works for workflows with no included workflows. If included
	// workflows are used, this func has to be called as wf.postValidateModifier.
	//postValidateWorkflowModifier(w, params)

	daisycommon.SetWorkflowAttributes(w, params.DaisyAttrs())
	return w, nil
}

func postValidateWorkflowModifier(w *daisy.Workflow, params *ovfexportdomain.OVFExportArgs) {
	rl := &daisyutils.ResourceLabeler{
		BuildID:         params.BuildID,
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
	daisyutils.UpdateAllInstanceNoExternalIP(w, params.NoExternalIP)
}

//TODO: consolidate with gce_vm_image_import.runStep()
func (oe *OVFExporter) runStep(ctx context.Context, step func() error, cancel func(string) bool, getTraceLogs func() []string) (err error) {
	e := make(chan error)
	var wg sync.WaitGroup
	go func() {
		//this select checks if context expired prior to runStep being called
		//if not, step is run
		select {
		case <-ctx.Done():
			e <- oe.getCtxError(ctx)
		default:
			wg.Add(1)
			var stepErr error
			defer func() {
				// error should only be returned after wg is marked as done. Otherwise,
				// a deadlock can occur when handling a timeout in the select below
				// because cancel() causes step() to finish, then waits for wg, while
				// writing to error chan waits on error chan reader which never happens
				wg.Done()
				e <- stepErr
			}()
			stepErr = step()
		}
	}()

	// this select waits for either context expiration or step to finish (with either an error or success)
	select {
	case <-ctx.Done():
		if cancel("timed-out") {
			//Only return timeout error if step was able to cancel on time-out.
			//Otherwise, step has finished and export succeeded even though it timed out
			err = oe.getCtxError(ctx)
		}
		wg.Wait()
	case stepErr := <-e:
		err = stepErr
	}
	oe.traceLogs = append(oe.traceLogs, getTraceLogs()...)
	return err
}

func (oe *OVFExporter) appendTraceLogs(traceLogs []string) {
	if traceLogs != nil && len(traceLogs) > 0 {
		oe.traceLogs = append(oe.traceLogs, traceLogs...)
	}
}

//TODO: consolidate with gce_vm_image_import.getCtxError()
func (oe *OVFExporter) getCtxError(ctx context.Context) (err error) {
	if ctxErr := ctx.Err(); ctxErr == context.DeadlineExceeded {
		err = daisy.Errf("OVF Export did not complete within the specified timeout of %s", oe.params.Timeout.String())
	} else {
		err = ctxErr
	}
	return err
}

func isInstanceRunning(instance *compute.Instance) bool {
	return !(instance == nil || instance.Status == "STOPPED" || instance.Status == "STOPPING" ||
		instance.Status == "SUSPENDED" || instance.Status == "SUSPENDING")
}

func getBootDisk(exportedDisks []*ovfexportdomain.ExportedDisk) *ovfexportdomain.ExportedDisk {
	for _, exportedDisk := range exportedDisks {
		if exportedDisk.AttachedDisk.Boot {
			return exportedDisk
		}
	}
	return nil
}
