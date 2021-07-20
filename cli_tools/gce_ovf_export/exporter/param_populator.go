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
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	pathutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
)

type ovfExportParamPopulatorImpl struct {
	param.Populator
}

// NewPopulator returns an object that implements OvfExportParamPopulator.
func NewPopulator(
	computeClient compute.Client,
	metadataClient domain.MetadataGCEInterface,
	storageClient domain.StorageClientInterface,
	resourceLocationRetriever domain.ResourceLocationRetrieverInterface,
	scratchBucketCreator domain.ScratchBucketCreatorInterface) ovfexportdomain.OvfExportParamPopulator {
	return &ovfExportParamPopulatorImpl{
		Populator: param.NewPopulator(param.NewNetworkResolver(computeClient), metadataClient, storageClient, resourceLocationRetriever, scratchBucketCreator),
	}
}

func (populator *ovfExportParamPopulatorImpl) Populate(params *ovfexportdomain.OVFExportArgs) (err error) {
	if err := populator.PopulateMissingParameters(&params.Project, params.ClientID, &params.Zone,
		&params.Region, &params.ScratchBucketGcsPath, params.DestinationURI, nil, &params.Network, &params.Subnet); err != nil {
		return err
	}
	populator.populateBuildID(params)
	populator.populateNamespacedScratchDirectory(params)
	populator.populateDestinationURI(params)
	return nil
}

func (populator *ovfExportParamPopulatorImpl) populateBuildID(params *ovfexportdomain.OVFExportArgs) {
	if params != nil && params.BuildID != "" {
		return
	}
	params.BuildID = os.Getenv("BUILD_ID")
	if params.BuildID == "" {
		params.BuildID = pathutils.RandString(5)
	}
}

func (populator *ovfExportParamPopulatorImpl) populateDestinationURI(params *ovfexportdomain.OVFExportArgs) {
	_, objectPath, err := storage.SplitGCSPath(params.DestinationURI)
	if err != nil {
		panic("params.DestinationURI should be validated before calling populate")
	}
	if objectPath != "" && strings.HasSuffix(strings.ToLower(objectPath), ".ovf") {
		if lastSlashIndex := strings.LastIndex(params.DestinationURI, "/"); lastSlashIndex > -1 {
			params.DestinationDirectory = pathutils.ToDirectoryURL(params.DestinationURI[:lastSlashIndex])
			// get the file name of the descriptor and use it for all the exported files
			params.OvfName = params.DestinationURI[lastSlashIndex+1 : len(params.DestinationURI)-4]
			return
		}
	}
	params.DestinationDirectory = pathutils.ToDirectoryURL(params.DestinationURI)
	params.OvfName = params.GetResourceName()
}

// populateNamespacedScratchDirectory updates ScratchBucketGcsPath to include a directory
// that is specific to this export, formulated using the start timestamp and the execution ID.
// This ensures all logs and artifacts are contained in a single directory.
func (populator *ovfExportParamPopulatorImpl) populateNamespacedScratchDirectory(params *ovfexportdomain.OVFExportArgs) {
	if !strings.HasSuffix(params.ScratchBucketGcsPath, "/") {
		params.ScratchBucketGcsPath += "/"
	}

	params.ScratchBucketGcsPath += fmt.Sprintf(
		"gce-ovf-export-%s-%s", params.Started.Format(time.RFC3339), params.BuildID)
}
