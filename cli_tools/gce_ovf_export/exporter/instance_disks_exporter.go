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
	"path"
	"strconv"
	"strings"

	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	stringutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/string"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

type instanceDisksExporterImpl struct {
	worker           daisyutils.DaisyWorker
	computeClient    daisycompute.Client
	storageClient    domain.StorageClientInterface
	exportedDisks    []*ovfexportdomain.ExportedDisk
	logger           logging.Logger
	wfPreRunCallback wfCallback
}

// NewInstanceDisksExporter creates a new instance disk exporter
func NewInstanceDisksExporter(computeClient daisycompute.Client, storageClient domain.StorageClientInterface, logger logging.Logger) ovfexportdomain.InstanceDisksExporter {
	return &instanceDisksExporterImpl{
		computeClient: computeClient,
		storageClient: storageClient,
		logger:        logger,
	}
}

func (ide *instanceDisksExporterImpl) Export(instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) ([]*ovfexportdomain.ExportedDisk, error) {
	wfName := "ovf-export-disk-export"
	workflowProvider := func() (wf *daisy.Workflow, err error) {
		if wf, err = generateWorkflowWithSteps(wfName, params.Timeout.String(),
			func(w *daisy.Workflow) error { return ide.populateExportDisksSteps(w, instance, params) }); err != nil {
			return nil, err
		}
		if ide.wfPreRunCallback != nil {
			ide.wfPreRunCallback(wf)
		}
		return wf, err
	}

	ide.worker = daisyutils.NewDaisyWorker(workflowProvider, params.EnvironmentSettings(wfName), ide.logger)
	if err := ide.worker.Run(map[string]string{}); err != nil {
		return nil, err
	}
	if err := ide.populateExportedDisksMetadata(params); err != nil {
		return nil, err
	}
	return ide.exportedDisks, nil
}

func (ide *instanceDisksExporterImpl) populateExportedDisksMetadata(params *ovfexportdomain.OVFExportArgs) error {
	// populate exported disks with compute.Disk references and storage object attributes
	for _, exportedDisk := range ide.exportedDisks {
		// populate compute.Disk for each exported disk
		if disk, err := ide.computeClient.GetDisk(params.Project, params.Zone, daisyutils.GetResourceID(exportedDisk.AttachedDisk.Source)); err == nil {
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

func (ide *instanceDisksExporterImpl) populateExportDisksSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) error {
	var err error
	ide.exportedDisks, err = ide.addExportDisksSteps(w, instance, params)
	if err != nil {
		return err
	}
	return nil
}

// addExportDisksSteps adds Daisy steps to OVF export workflow to export disks.
// It returns an array of GCS paths of exported disks in the same order as Instance.Disks.
func (ide *instanceDisksExporterImpl) addExportDisksSteps(w *daisy.Workflow, instance *compute.Instance, params *ovfexportdomain.OVFExportArgs) ([]*ovfexportdomain.ExportedDisk, error) {
	if instance == nil || len(instance.Disks) == 0 {
		return nil, daisy.Errf("No attachedDisks found in the Instance to export")
	}
	attachedDisks := instance.Disks
	var exportedDisks []*ovfexportdomain.ExportedDisk

	for i, attachedDisk := range attachedDisks {
		indexOfProjects := strings.Index(attachedDisk.Source, "projects/")
		if indexOfProjects < 0 {
			return nil, daisy.Errf("Disk source `%v` is invalid.", attachedDisk.Source)
		}
		diskPath := attachedDisk.Source[indexOfProjects:]
		var exportedDiskFileName string
		if strings.HasPrefix(attachedDisk.DeviceName, params.OvfName) {
			exportedDiskFileName = fmt.Sprintf("%v.%v", attachedDisk.DeviceName, params.DiskExportFormat)
		} else {
			exportedDiskFileName = fmt.Sprintf("%v-%v.%v", params.OvfName, attachedDisk.DeviceName, params.DiskExportFormat)
		}
		exportedDiskGCSPath := fmt.Sprintf("%v%v", params.DestinationDirectory, exportedDiskFileName)
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

		varMap := map[string]string{
			"source_disk":                diskPath,
			"destination":                exportedDiskGCSPath,
			"format":                     params.DiskExportFormat,
			"export_instance_disk_image": "projects/compute-image-tools/global/images/family/debian-9-worker",
			"export_instance_disk_size":  "200",
			"export_network":             params.Network,
			"export_subnet":              params.Subnet,
			"export_disk_ext.sh":         "../export/export_disk_ext.sh",
			"disk_resizing_mon.sh":       "../export/disk_resizing_mon.sh",
		}
		if params.ComputeServiceAccount != "" {
			varMap["compute_service_account"] = params.ComputeServiceAccount
		}
		var err daisy.DError
		exportDiskStep.IncludeWorkflow, err = instantiateIncludedWorkflow(w, path.Join(params.WorkflowDir, "/export/disk_export_ext.wf.json"), varMap)
		if err != nil {
			return nil, err
		}
		w.Steps[exportDiskStepName] = exportDiskStep
	}
	return exportedDisks, nil
}

// instantiateIncludedWorkflow creates an included workflow from the JSON file includedWorkflowPath,
// using the workflow w as its parent, and applying varMap as the variables.
func instantiateIncludedWorkflow(w *daisy.Workflow, includedWorkflowPath string,
	varMap map[string]string) (*daisy.IncludeWorkflow, daisy.DError) {
	iw := &daisy.IncludeWorkflow{
		Path: includedWorkflowPath,
		Vars: varMap,
	}
	var err daisy.DError
	iw.Workflow, err = w.NewIncludedWorkflowFromFile(includedWorkflowPath)
	if err != nil {
		return nil, err
	}
	return iw, nil
}

func (ide *instanceDisksExporterImpl) Cancel(reason string) bool {
	if ide.worker == nil {
		return false
	}
	ide.worker.Cancel(reason)
	return true
}
