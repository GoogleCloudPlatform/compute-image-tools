//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

// GetProjectID gets project id from flag if exists; otherwise, try to retrieve from GCE metadata.
func GetProjectID(mgce domain.MetadataGCEInterface, projectFlag string) (string, error) {
	if projectFlag == "" {
		if !mgce.OnGCE() {
			return "", fmt.Errorf("project cannot be determined because build is not running on GCE")
		}
		aProject, err := mgce.ProjectID()
		if err != nil || aProject == "" {
			return "", fmt.Errorf("project cannot be determined %v", err)
		}
		return aProject, nil
	}
	return projectFlag, nil
}

// PopulateMissingParameters populate missing params for import/export cli tools
func PopulateMissingParameters(project *string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, mgce domain.MetadataGCEInterface,
	scratchBucketCreator domain.ScratchBucketCreatorInterface,
	zoneRetriever domain.ZoneRetrieverInterface,
	storageClient domain.StorageClientInterface) error {

	if err := PopulateProjectIfMissing(mgce, project); err != nil {
		return err
	}

	scratchBucketRegion, err := populateBucket(scratchBucketGcsPath, zone, mgce, scratchBucketCreator,
		file, project, storageClient)
	if err != nil {
		return err
	}

	if *zone == "" {
		if aZone, err := zoneRetriever.GetZone(scratchBucketRegion, *project); err == nil {
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

func populateBucket(scratchBucketGcsPath *string, zone *string, mgce domain.MetadataGCEInterface,
	scratchBucketCreator domain.ScratchBucketCreatorInterface, file string, project *string,
	storageClient domain.StorageClientInterface) (string, error) {

	scratchBucketRegion := ""
	if *scratchBucketGcsPath == "" {
		fallbackZone := *zone
		if fallbackZone == "" && mgce.OnGCE() {
			var err error
			if fallbackZone, err = mgce.Zone(); err != nil {
				// reset fallback zone if failed to get zone from running GCE
				fallbackZone = ""
			}
		}

		scratchBucketName, sbr, err := scratchBucketCreator.CreateScratchBucket(file, *project, fallbackZone)
		scratchBucketRegion = sbr
		if err != nil {
			return "", err
		}

		*scratchBucketGcsPath = fmt.Sprintf("gs://%v/", scratchBucketName)
	} else {
		scratchBucketName, err := storage.GetBucketNameFromGCSPath(*scratchBucketGcsPath)
		if err != nil {
			return "", fmt.Errorf("invalid scratch bucket GCS path %v", scratchBucketGcsPath)

		}

		scratchBucketAttrs, err := storageClient.GetBucketAttrs(scratchBucketName)
		if err == nil {
			scratchBucketRegion = scratchBucketAttrs.Location
		}
	}
	return scratchBucketRegion, nil
}

// PopulateProjectIfMissing populates project id for cli tools
func PopulateProjectIfMissing(mgce domain.MetadataGCEInterface, projectFlag *string) error {
	var err error
	*projectFlag, err = GetProjectID(mgce, *projectFlag)
	return err
}

// PopulateRegion populates region based on the value extracted from zone param
func PopulateRegion(region *string, zone string) error {
	aRegion, err := paramhelper.GetRegion(zone)
	if err != nil {
		return err
	}
	*region = aRegion
	return nil
}

// CreateComputeClient creates a new compute client
func CreateComputeClient(ctx *context.Context, oauth string, ce string) compute.Client {
	computeOptions := []option.ClientOption{option.WithCredentialsFile(oauth)}
	if ce != "" {
		computeOptions = append(computeOptions, option.WithEndpoint(ce))
	}

	computeClient, err := compute.NewClient(*ctx, computeOptions...)
	if err != nil {
		log.Fatalf("compute client: %v", err)
	}
	return computeClient
}
