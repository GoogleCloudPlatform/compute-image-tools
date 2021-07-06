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

package storage

import (
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

type multiRegion struct {
	regions                      []string
	isRegionToMultiRegionAllowed bool
}

var (
	// for each GCS multi region, regions are sorted by GCE cost in ascending order as of the time of writing
	// this code.
	// GCE cost per region: https://cloud.google.com/compute/vm-instance-pricing
	// GCS multi-regions: https://cloud.google.com/storage/docs/locations#location-mr
	multiRegions = map[string]multiRegion{
		"US":   {[]string{"us-central1", "us-east1", "us-west1", "us-east4", "us-west4", "us-west2", "us-west3"}, true},
		"EU":   {[]string{"europe-west1", "europe-west4", "europe-north1", "europe-west2", "europe-west3", "europe-west6"}, true},
		"ASIA": {[]string{"asia-east1", "asia-south1", "asia-southeast1", "asia-northeast1", "asia-northeast2", "asia-northeast3", "asia-southeast2", "asia-east2"}, true},

		"EUR4": {[]string{"europe-west4", "europe-north1"}, false},
		"NAM4": {[]string{"us-central1", "us-east1"}, false},
	}
)

// ResourceLocationRetriever is responsible for retrieving GCE zone to run import in
type ResourceLocationRetriever struct {
	Mgce              domain.MetadataGCEInterface
	ComputeGCEService daisycompute.Client
}

// NewResourceLocationRetriever creates a ResourceLocationRetriever
func NewResourceLocationRetriever(aMgce domain.MetadataGCEInterface, cs daisycompute.Client) *ResourceLocationRetriever {
	return &ResourceLocationRetriever{Mgce: aMgce, ComputeGCEService: cs}
}

// GetZone retrieves GCE zone to run import in based on imported source file location and available
// zones in the project.
// If storageRegion is provided and valid, first zone within that region will be used.
// If no storageRegion is provided, GCE Zone from the running process
// will be used.
func (rlr *ResourceLocationRetriever) GetZone(storageLocation string, project string) (string, error) {
	zone := ""
	var err error
	if storageLocation != "" {
		// pick a zone from the region where data is stored
		zone, err = rlr.getZoneFromStorageLocation(storageLocation, project)
		if err == nil {
			return zone, err
		}
	}

	// determine zone based on the zone Cloud Build is running in
	if rlr.Mgce.OnGCE() {
		zone, err = rlr.Mgce.Zone()
	}
	if err != nil {
		return "", daisy.Errf("can't infer zone: %v", err)
	}
	if zone == "" {
		return "", daisy.Errf("zone is empty")
	}
	fmt.Printf("[image-import] Zone not provided, using %v\n", zone)

	return zone, nil
}

func (rlr *ResourceLocationRetriever) getZoneFromStorageLocation(location string, project string) (string, daisy.DError) {
	if project == "" {
		return "", daisy.Errf("project cannot be empty in order to find a zone from a location")
	}
	zones, err := rlr.ComputeGCEService.ListZones(project)
	if err != nil {
		return "", daisy.Errf("Failed to list zones: %v", err)
	}
	if rlr.isMultiRegion(location) {
		return rlr.getBestZoneForMultiRegion(location, zones)
	}
	return rlr.getZoneForRegion(location, zones)
}

func (rlr *ResourceLocationRetriever) getZoneForRegion(region string, zones []*compute.Zone) (string, daisy.DError) {
	for _, zone := range zones {
		if isZoneUp(zone) && strings.HasSuffix(strings.ToLower(zone.Region), strings.ToLower(region)) {
			return zone.Name, nil
		}
	}
	return "", daisy.Errf("no zone found for %v region", region)
}

func (rlr *ResourceLocationRetriever) getMultiRegionForRegion(region string) string {
	for multiRegionID, multiRegion := range multiRegions {
		if !multiRegion.isRegionToMultiRegionAllowed {
			continue
		}

		for _, aRegion := range multiRegion.regions {
			if strings.EqualFold(aRegion, region) {
				return multiRegionID
			}
		}
	}

	return ""
}

func (rlr *ResourceLocationRetriever) getBestZoneForMultiRegion(multiRegion string, zones []*compute.Zone) (string, daisy.DError) {
	for _, region := range multiRegions[multiRegion].regions {
		if zone, err := rlr.getZoneForRegion(region, zones); err == nil {
			return zone, nil
		}
	}
	return "", daisy.Errf("no zones found for %v multi region", multiRegion)
}

func (rlr *ResourceLocationRetriever) isMultiRegion(location string) bool {
	_, containsKey := multiRegions[strings.ToUpper(location)]
	return containsKey
}

func isZoneUp(zone *compute.Zone) bool {
	return zone != nil && zone.Status == "UP"
}

// GetLargestStorageLocation returns the largest storage location that includes provided argument.
// If argument is a multi-region, the argument is returned. If argument is a region within a multi-region,
// the multi-region is returned. If argument is a region not within a multi-region, argument is returned.
func (rlr *ResourceLocationRetriever) GetLargestStorageLocation(storageLocation string) string {
	if rlr.isMultiRegion(storageLocation) {
		return storageLocation
	}

	if multiRegion := rlr.getMultiRegionForRegion(storageLocation); multiRegion != "" {
		return multiRegion
	}

	return storageLocation
}
