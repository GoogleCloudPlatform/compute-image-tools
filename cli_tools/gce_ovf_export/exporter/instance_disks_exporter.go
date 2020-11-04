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
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

type instanceDisksExporterImpl struct {
	wf            *daisy.Workflow
	computeClient daisycompute.Client
	storageClient domain.StorageClientInterface
	exportedDisks []*ovfexportdomain.ExportedDisk
	serialLogs    []string
	wfCallback    wfCallback
}

// NewInstanceDisksExporter creates a new instance disk exporter
func NewInstanceDisksExporter(computeClient daisycompute.Client, storageClient domain.StorageClientInterface) ovfexportdomain.InstanceDisksExporter {
	return &instanceDisksExporterImpl{
		computeClient: computeClient,
		storageClient: storageClient,
	}
}

type wfCallback func(w *daisy.Workflow)

func (ide *instanceDisksExporterImpl) Export(instance *compute.Instance, params *ovfexportdomain.OVFExportParams) ([]*ovfexportdomain.ExportedDisk, error) {
	var err error
	if ide.wf, err = generateWorkflowWithSteps("ovf-export-disk-export", params.Timeout.String(),
		func(w *daisy.Workflow) error { return ide.populateExportDisksSteps(w, instance, params) }, params); err != nil {
		return nil, err
	}
	if ide.wfCallback != nil {
		ide.wfCallback(ide.wf)
	}
	if err := daisyutils.RunWorkflowWithCancelSignal(context.Background(), ide.wf); err != nil {
		return nil, err
	}
	// have to use post-validate modifiers due to the use of included workflows
	//err = ide.wf.RunWithModifiers(context.Background(), nil, func(w *daisy.Workflow) {
	//	postValidateWorkflowModifier(w, params)
	//})
	if ide.wf.Logger != nil {
		ide.serialLogs = ide.wf.Logger.ReadSerialPortLogs()
	}
	if err := ide.populateExportedDisksMetadata(params); err != nil {
		return nil, err
	}
	return ide.exportedDisks, err
}

func (ide *instanceDisksExporterImpl) populateExportedDisksMetadata(params *ovfexportdomain.OVFExportParams) error {
	// populate exported disks with compute.Disk references and storage object attributes
	for _, exportedDisk := range ide.exportedDisks {
		// populate compute.Disk for each exported disk
		if disk, err := ide.computeClient.GetDisk(*params.Project, params.Zone, daisyutils.GetResourceID(exportedDisk.AttachedDisk.Source)); err == nil {
			exportedDisk.Disk = disk
		} else {
			return err
		}

		// populate storage object attributes for each exported disk file
		bucketName, objectPath, err := storageutils.SplitGCSPath(exportedDisk.GcsPath)
		if err != nil {
			return err
		}
		exportedDisk.GcsFileAttrs, err = ide.storageClient.GetObjectAttrs(bucketName, objectPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ide *instanceDisksExporterImpl) populateExportDisksSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportParams) error {
	var err error
	ide.exportedDisks, err = ide.addExportDisksSteps(w, instance, []string{}, params)
	if err != nil {
		return err
	}
	return nil
}

// addExportDisksSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (ide *instanceDisksExporterImpl) addExportDisksSteps(w *daisy.Workflow, instance *compute.Instance, previousStepNames []string, params *ovfexportdomain.OVFExportParams) ([]*ovfexportdomain.ExportedDisk, error) {
	if instance == nil || len(instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks
	var exportedDisks []*ovfexportdomain.ExportedDisk

	for i, attachedDisk := range attachedDisks {
		diskPath := attachedDisk.Source[strings.Index(attachedDisk.Source, "projects/"):]
		exportedDiskGCSPath := params.DestinationURI + attachedDisk.DeviceName + "." + params.DiskExportFormat
		exportedDisks = append(exportedDisks, &ovfexportdomain.ExportedDisk{AttachedDisk: attachedDisk, GcsPath: exportedDiskGCSPath})

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
			Path: params.WorkflowDir + "/export/disk_export_ext.wf.json",
			Vars: map[string]string{
				"source_disk":                diskPath,
				"destination":                exportedDiskGCSPath,
				"format":                     params.DiskExportFormat,
				"export_instance_disk_image": "projects/compute-image-tools/global/images/family/debian-9-worker",
				"export_instance_disk_size":  "200",
				"export_instance_disk_type":  "pd-ssd",
				"export_network":             params.Network,
				"export_subnet":              params.Subnet,
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
	}
	return exportedDisks, nil
}

func (ide *instanceDisksExporterImpl) TraceLogs() []string {
	return ide.serialLogs
}

func (ide *instanceDisksExporterImpl) Cancel(reason string) bool {
	if ide.wf == nil {
		return false
	}
	ide.wf.CancelWithReason(reason)
	return true
}
