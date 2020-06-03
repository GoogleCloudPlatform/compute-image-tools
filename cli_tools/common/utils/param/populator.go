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

package param

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
)

// Populator standardizes user input, and determines omitted values.
type Populator interface {
	PopulateMissingParameters(project *string, zone *string, region *string,
		scratchBucketGcsPath *string, file string, storageLocation *string) error
}

// NewPopulator returns an object that implements Populator.
func NewPopulator(
	metadataClient domain.MetadataGCEInterface,
	storageClient domain.StorageClientInterface,
	locationClient domain.ResourceLocationRetrieverInterface,
	scratchBucketClient domain.ScratchBucketCreatorInterface) Populator {
	return populator{
		metadataClient:      metadataClient,
		storageClient:       storageClient,
		locationClient:      locationClient,
		scratchBucketClient: scratchBucketClient,
	}
}

type populator struct {
	metadataClient      domain.MetadataGCEInterface
	storageClient       domain.StorageClientInterface
	locationClient      domain.ResourceLocationRetrieverInterface
	scratchBucketClient domain.ScratchBucketCreatorInterface
}

func (p populator) PopulateMissingParameters(project *string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, storageLocation *string) error {

	if err := PopulateProjectIfMissing(p.metadataClient, project); err != nil {
		return err
	}

	scratchBucketRegion, err := populateScratchBucketGcsPath(scratchBucketGcsPath, *zone, p.metadataClient,
		p.scratchBucketClient, file, project, p.storageClient)
	if err != nil {
		return err
	}

	if storageLocation != nil && *storageLocation == "" {
		*storageLocation = p.locationClient.GetLargestStorageLocation(scratchBucketRegion)
	}

	if *zone == "" {
		if aZone, err := p.locationClient.GetZone(scratchBucketRegion, *project); err == nil {
			*zone = aZone
		} else {
			return err
		}
	}

	if err := PopulateRegion(region, *zone); err != nil {
		return err
	}
	return nil
}
