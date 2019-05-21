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

package paramutils

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/option"
)

// GetProjectID gets project id from flag if exists; otherwise, try to retrieve from GCE metadata.
func GetProjectID(mgce commondomain.MetadataGCEInterface, projectFlag string) (string, error) {
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
	scratchBucketGcsPath *string, file string, mgce commondomain.MetadataGCEInterface,
	scratchBucketCreator commondomain.ScratchBucketCreatorInterface,
	zoneRetriever commondomain.ZoneRetrieverInterface,
	storageClient commondomain.StorageClientInterface) error {

	if err := PopulateProjectIfMissing(mgce, project); err != nil {
		return err
	}

	scratchBucketRegion := ""
	if *scratchBucketGcsPath == "" {
		scratchBucketName, sbr, err := scratchBucketCreator.CreateScratchBucket(file, *project)
		scratchBucketRegion = sbr
		if err != nil {
			return err
		}

		*scratchBucketGcsPath = fmt.Sprintf("gs://%v/", scratchBucketName)
	} else {
		scratchBucketName, err := storageutils.GetBucketNameFromGCSPath(*scratchBucketGcsPath)
		if err != nil {
			return fmt.Errorf("invalid scratch bucket GCS path %v", *scratchBucketGcsPath)
		}
		scratchBucketAttrs, err := storageClient.GetBucketAttrs(scratchBucketName)
		if err == nil {
			scratchBucketRegion = scratchBucketAttrs.Location
		}
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

// PopulateProjectIfMissing populates project id for cli tools
func PopulateProjectIfMissing(mgce commondomain.MetadataGCEInterface, projectFlag *string) error {
	var err error
	*projectFlag, err = GetProjectID(mgce, *projectFlag)
	return err
}

// PopulateRegion populates region for cli tools
func PopulateRegion(region *string, zone string) error {
	aRegion, err := GetRegion(zone)
	if err != nil {
		return err
	}
	*region = aRegion
	return nil
}

// GetRegion gets region by zone
func GetRegion(zone string) (string, error) {
	if zone == "" {
		return "", fmt.Errorf("zone is empty. Can't determine region")
	}
	zoneStrs := strings.Split(zone, "-")
	if len(zoneStrs) < 2 {
		return "", fmt.Errorf("%v is not a valid zone", zone)
	}
	return strings.Join(zoneStrs[:len(zoneStrs)-1], "-"), nil
}

// CreateComputeClient creates a new Daisy Compute client
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
