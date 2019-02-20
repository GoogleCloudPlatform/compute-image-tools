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

package gcevmimageimportutil

import (
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_vm_image_import/domain"
	"strings"
)

// ZoneRetriever is responsible for retrieving GCE zone to run import in
type ZoneRetriever struct {
	Mgce              domain.MetadataGCEInterface
	ComputeGCEService domain.ComputeServiceInterface
}

// NewZoneRetriever creates a ZoneRetriever
func NewZoneRetriever(aMgce domain.MetadataGCEInterface, cs domain.ComputeServiceInterface) (*ZoneRetriever, error) {
	return &ZoneRetriever{Mgce: aMgce, ComputeGCEService: cs}, nil
}

// GetZone retrieves GCE zone to run import in based on imported source file region and available
// zones in the project.
// If storageRegion is provided and valid, first zone within that region will be used.
// If no storageRegion is provided, GCE Zone from the running process
// will be used.
func (zr *ZoneRetriever) GetZone(storageRegion string, project string) (string, error) {
	zone := ""
	var err error
	if storageRegion != "" {
		// pick a random zone from the region where data is stored
		zone, err = zr.getZoneFromRegion(storageRegion, project)
		if err == nil {
			return zone, err
		}
	}

	// determine zone based on the zone Cloud Build is running in
	if zr.Mgce.OnGCE() {
		zone, err = zr.Mgce.Zone()
	}

	if err != nil {
		return "", fmt.Errorf("can't infer zone: %v", err)
	}
	if zone == "" {
		return "", fmt.Errorf("zone is empty")
	}
	fmt.Printf("[image-importer] Zone not provided, using %v\n", zone)

	return zone, nil
}

func (zr *ZoneRetriever) getZoneFromRegion(region string, project string) (string, error) {
	if project == "" {
		return "", fmt.Errorf("project cannot be empty in order to find a zone from a region")
	}
	zones, err := zr.ComputeGCEService.GetZones(project)
	if err != nil {
		return "", err
	}
	for _, zone := range zones {
		if strings.HasSuffix(strings.ToLower(zone.Region), strings.ToLower(region)) {
			return zone.Name, nil
		}
	}
	return "", fmt.Errorf("No zone found for %v region", region)
}
