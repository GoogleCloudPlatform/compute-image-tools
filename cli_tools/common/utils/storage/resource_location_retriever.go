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

var (
	// for each multi region, regions are sorted by cost in ascending order as of the time of writing
	// this code.
	multiRegions = map[string][]string{
		"US":   {"us-central1", "us-east1", "us-west1", "us-east4", "us-west2"},
		"EU":   {"europe-north1", "europe-west1", "europe-west4", "europe-west2", "europe-west3"},
		"ASIA": {"asia-east1", "asia-south1", "asia-southeast1", "asia-northeast1", "asia-east2"},
		"EUR4": {"europe-north1", "europe-west4"},
		"NAM4": {"us-central1", "us-east1"},
	}
)

// ZoneRetriever is responsible for retrieving GCE zone to run import in
type ZoneRetriever struct {
	Mgce              domain.MetadataGCEInterface
	ComputeGCEService daisycompute.Client
}

// NewZoneRetriever creates a ZoneRetriever
func NewZoneRetriever(aMgce domain.MetadataGCEInterface, cs daisycompute.Client) *ZoneRetriever {
	return &ZoneRetriever{Mgce: aMgce, ComputeGCEService: cs}
}

// GetZone retrieves GCE zone to run import in based on imported source file location and available
// zones in the project.
// If storageRegion is provided and valid, first zone within that region will be used.
// If no storageRegion is provided, GCE Zone from the running process
// will be used.
func (zr *ZoneRetriever) GetZone(storageLocation string, project string) (string, error) {
	zone := ""
	var err error
	if storageLocation != "" {
		// pick a zone from the region where data is stored
		zone, err = zr.getZoneFromStorageLocation(storageLocation, project)
		if err == nil {
			return zone, err
		}
	}

	// determine zone based on the zone Cloud Build is running in
	if zr.Mgce.OnGCE() {
		zone, err = zr.Mgce.Zone()
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

func (zr *ZoneRetriever) getZoneFromStorageLocation(location string, project string) (string, daisy.DError) {
	if project == "" {
		return "", daisy.Errf("project cannot be empty in order to find a zone from a location")
	}
	zones, err := zr.ComputeGCEService.ListZones(project)
	if err != nil {
		return "", daisy.Errf("Failed to list zones: %v", err)
	}
	if zr.isMultiRegion(location) {
		return zr.getBestZoneForMultiRegion(location, zones)
	}
	return zr.getZoneForRegion(location, zones)
}

func (zr *ZoneRetriever) getZoneForRegion(region string, zones []*compute.Zone) (string, daisy.DError) {
	for _, zone := range zones {
		if isZoneUp(zone) && strings.HasSuffix(strings.ToLower(zone.Region), strings.ToLower(region)) {
			return zone.Name, nil
		}
	}
	return "", daisy.Errf("no zone found for %v region", region)
}

func (zr *ZoneRetriever) getBestZoneForMultiRegion(multiRegion string, zones []*compute.Zone) (string, daisy.DError) {
	for _, region := range multiRegions[multiRegion] {
		if zone, err := zr.getZoneForRegion(region, zones); err == nil {
			return zone, nil
		}
	}
	return "", daisy.Errf("no zones found for %v multi region", multiRegion)
}

func (zr *ZoneRetriever) isMultiRegion(location string) bool {
	_, containsKey := multiRegions[location]
	return containsKey
}

func isZoneUp(zone *compute.Zone) bool {
	return zone != nil && zone.Status == "UP"
}
