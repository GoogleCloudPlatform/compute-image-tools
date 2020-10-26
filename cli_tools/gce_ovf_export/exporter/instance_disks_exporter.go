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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	daisyutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

// InstanceDisksExporter exports disks of an instance
type InstanceDisksExporter interface {
	Export(*compute.Instance) ([]*ExportedDisk, error)
	TraceLogs() []string
	Cancel(reason string) bool
}

type instanceDisksExporterImpl struct {
	wf                *daisy.Workflow
	params            *OVFExportParams
	workflowGenerator *OVFExportWorkflowGenerator
	computeClient     daisycompute.Client
	storageClient     domain.StorageClientInterface
	exportedDisks     []*ExportedDisk
	workflowPath      string
	serialLogs        []string
}

// NewInstanceDisksExporter creates a new instance disk exporter
func NewInstanceDisksExporter(params *OVFExportParams, workflowGenerator *OVFExportWorkflowGenerator,
	workflowPath string, computeClient daisycompute.Client, storageClient domain.StorageClientInterface) InstanceDisksExporter {
	return &instanceDisksExporterImpl{
		params:            params,
		workflowGenerator: workflowGenerator,
		workflowPath:      workflowPath,
		computeClient:     computeClient,
		storageClient:     storageClient,
	}
}

func (ide *instanceDisksExporterImpl) Export(instance *compute.Instance) ([]*ExportedDisk, error) {
	var err error
	ide.wf, err = runWorkflowWithSteps(context.Background(),
		"ovf-export-disk-export", ide.workflowPath, ide.params.Timeout,
		func(w *daisy.Workflow) error { return ide.populateExportDisksSteps(w, instance) }, map[string]string{}, ide.params)
	if ide.wf.Logger != nil {
		ide.serialLogs = ide.wf.Logger.ReadSerialPortLogs()
	}
	if err := ide.populateExportedDisksMetadata(); err != nil {
		return nil, err
	}
	return ide.exportedDisks, err
}

func (ide *instanceDisksExporterImpl) populateExportedDisksMetadata() error {
	// populate exported disks with compute.Disk references and storage object attributes
	for _, exportedDisk := range ide.exportedDisks {
		// populate compute.Disk for each exported disk
		if disk, err := ide.computeClient.GetDisk(*ide.params.Project, ide.params.Zone, daisyutils.GetResourceID(exportedDisk.attachedDisk.Source)); err == nil {
			exportedDisk.disk = disk
		} else {
			return err
		}

		// populate storage object attributes for each exported disk file
		bucketName, objectPath, err := storageutils.SplitGCSPath(exportedDisk.gcsPath)
		if err != nil {
			return err
		}
		exportedDisk.gcsFileAttrs, err = ide.storageClient.GetObjectAttrs(bucketName, objectPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ide *instanceDisksExporterImpl) populateExportDisksSteps(w *daisy.Workflow, instance *compute.Instance) error {
	var err error
	ide.exportedDisks, err = ide.workflowGenerator.addExportDisksSteps(w, instance, []string{}, "")
	if err != nil {
		return err
	}
	return nil
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
