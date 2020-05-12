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

// Fixer standardizes user input, and determines omitted values.
type Fixer interface {
	PopulateMissingParameters(project *string, zone *string, region *string,
		scratchBucketGcsPath *string, file string, storageLocation *string) error
}

// NewFixer returns an object that implements Fixer.
func NewFixer(
	metadataClient domain.MetadataGCEInterface,
	storageClient domain.StorageClientInterface,
	locationClient domain.ResourceLocationRetrieverInterface,
	scratchBucketClient domain.ScratchBucketCreatorInterface) Fixer {
	return fixer{
		metadataClient:      metadataClient,
		storageClient:       storageClient,
		locationClient:      locationClient,
		scratchBucketClient: scratchBucketClient,
	}
}

type fixer struct {
	metadataClient      domain.MetadataGCEInterface
	storageClient       domain.StorageClientInterface
	locationClient      domain.ResourceLocationRetrieverInterface
	scratchBucketClient domain.ScratchBucketCreatorInterface
}

func (f fixer) PopulateMissingParameters(project *string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, storageLocation *string) error {

	if err := PopulateProjectIfMissing(f.metadataClient, project); err != nil {
		return err
	}

	scratchBucketRegion, err := populateScratchBucketGcsPath(scratchBucketGcsPath, *zone, f.metadataClient,
		f.scratchBucketClient, file, project, f.storageClient)
	if err != nil {
		return err
	}

	if storageLocation != nil && *storageLocation == "" {
		*storageLocation = f.locationClient.GetLargestStorageLocation(scratchBucketRegion)
	}

	if *zone == "" {
		if aZone, err := f.locationClient.GetZone(scratchBucketRegion, *project); err == nil {
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
