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

package importer

import (
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

// processor represents the second (and final) phase of import. For bootable
// disks, this means translation and publishing the final image. For data
// disks, this means publishing the image.
//
// Implementers can expose detailed logs using the traceLogs() method.
type processor interface {
	// Returns a pd with updated values. It can be a different pd with different URI.
	process(persistentDisk) (persistentDisk, error)
	cancel(reason string) bool
}

// processorProvider allows the processor to be determined after the pd has been inflated.
type processorProvider interface {
	provide(pd persistentDisk) ([]processor, error)
}

type defaultProcessorProvider struct {
	ImageImportRequest
	computeClient daisyCompute.Client
	planner       processPlanner
	logger        logging.Logger
}

func (d defaultProcessorProvider) provide(pd persistentDisk) ([]processor, error) {

	if d.DataDisk {
		return []processor{
			newDataDiskProcessor(pd, d.computeClient, d.Project,
				d.Labels, d.StorageLocation, d.Description,
				d.Family, d.ImageName)}, nil
	}

	plan, err := d.planner.plan(pd)
	if err != nil {
		return nil, err
	}

	var processors []processor
	if plan.metadataChangesRequired() {
		p := newMetadataProcessor(d.ImageImportRequest.Project, d.ImageImportRequest.Zone, d.computeClient)
		p.requiredLicenses = plan.requiredLicenses
		p.requiredFeatures = plan.requiredFeatures
		processors = append(processors, p)
	}

	bootableDiskProcessor := newBootableDiskProcessor(d.ImageImportRequest, plan.translationWorkflowPath, d.logger, plan.detectedOs)
	if err != nil {
		return nil, err
	}
	return append(processors, bootableDiskProcessor), nil
}
