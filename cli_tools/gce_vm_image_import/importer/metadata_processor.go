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
	"fmt"
	"log"
	"strings"

	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

// metadataProcessor ensures metadata is present on a GCP disk. If
// metadata is missing, the disk will be recreated.
type metadataProcessor struct {
	project, zone     string
	computeDiskClient daisyCompute.Client

	requiredLicenses []string
	requiredFeatures []*compute.GuestOsFeature
}

func newMetadataProcessor(
	project string,
	zone string,
	computeDiskClient daisyCompute.Client) *metadataProcessor {
	return &metadataProcessor{project: project, zone: zone, computeDiskClient: computeDiskClient}
}

func (p *metadataProcessor) process(pd persistentDisk,
	loggableBuilder *service.SingleImageImportLoggableBuilder) (persistentDisk, error) {

	// Fast path 1: No modification requested.
	if len(p.requiredFeatures) == 0 && len(p.requiredLicenses) == 0 {
		return pd, nil
	}

	// Fast path 2: The current disk already has the requested modifications present.
	newDisk, changesRequired, err := p.stageRequestForNewDisk(pd)
	if !changesRequired || err != nil {
		return pd, err
	}

	// Slow path: Clone then delete the existing disk and return a reference
	// to the new disk.
	err = p.computeDiskClient.CreateDisk(p.project, p.zone, newDisk)
	if err != nil {
		return pd, daisy.Errf("Failed to create UEFI disk: %v", err)
	}
	log.Println("UEFI disk created: ", newDisk.Name)

	// Delete the old disk after the new disk is created.
	deleteDisk(p.computeDiskClient, p.project, p.zone, pd)

	// Update the new disk URI
	pd.uri = fmt.Sprintf("zones/%v/disks/%v", p.zone, newDisk.Name)
	return pd, nil
}

// stageRequestForNewDisk fetches the disk associated with `pd`, and creates a compute.Disk
// struct that contains requested modifications.
//
// When false, `cloneRequired` signifies that the current disk satisfies all requested changes.
func (p *metadataProcessor) stageRequestForNewDisk(pd persistentDisk) (newDisk *compute.Disk, cloneRequired bool, err error) {

	diskName := daisyUtils.GetResourceID(pd.uri)
	currentDisk, err := p.computeDiskClient.GetDisk(p.project, p.zone, diskName)
	if err != nil {
		return nil, false, daisy.Errf("Failed to get disk: %v", err)
	}

	newDiskName := fmt.Sprintf("%v-1", diskName)
	newDisk = &compute.Disk{
		Name:       newDiskName,
		SourceDisk: pd.uri,
	}
	if len(currentDisk.GuestOsFeatures) > 0 {
		newDisk.GuestOsFeatures = make([]*compute.GuestOsFeature, len(currentDisk.GuestOsFeatures))
		copy(newDisk.GuestOsFeatures, currentDisk.GuestOsFeatures)
	}
	if len(currentDisk.Licenses) > 0 {
		newDisk.Licenses = make([]string, len(currentDisk.Licenses))
		copy(newDisk.Licenses, currentDisk.Licenses)
	}

	cloneRequired = false
	for _, feature := range p.requiredFeatures {
		if !hasGuestOSFeature(currentDisk, feature) {
			newDisk.GuestOsFeatures = append(newDisk.GuestOsFeatures, feature)
			cloneRequired = true
		}
	}
	for _, license := range p.requiredLicenses {
		if !hasLicense(currentDisk, license) {
			newDisk.Licenses = append(newDisk.Licenses, license)
			cloneRequired = true
		}
	}
	return newDisk, cloneRequired, nil
}

func hasLicense(existingDisk *compute.Disk, requestedLicense string) bool {
	for _, foundLicense := range existingDisk.Licenses {
		// Licenses are expressed as [GCP resources](https://cloud.google.com/apis/design/resource_names),
		// which can either be a full URL, or a path. Both of these are valid:
		//   -  https://www.googleapis.com/compute/v1/projects/windows-cloud/global/licenses/windows-10-enterprise-byol
		//   -  projects/windows-cloud/global/licenses/windows-10-enterprise-byol
		// The double suffix match supports all combinations of full URI and path.
		if strings.HasSuffix(foundLicense, requestedLicense) || strings.HasSuffix(requestedLicense, foundLicense) {
			return true
		}
	}
	return false
}

func hasGuestOSFeature(existingDisk *compute.Disk, feature *compute.GuestOsFeature) bool {
	for _, f := range existingDisk.GuestOsFeatures {
		if f.Type == feature.Type {
			return true
		}
	}
	return false
}

func (p *metadataProcessor) cancel(reason string) bool {
	// Cancel is not performed since there is only one critical API call - CreateDisk
	return false
}

func (p *metadataProcessor) traceLogs() []string {
	return []string{}
}
