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

	daisyUtils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

// metadataProcessor ensures metadata is present on a GCP disk. If
// metadata is missing, the disk will be recreated.
type metadataProcessor struct {
	args              ImportArguments
	computeDiskClient daisyCompute.Client
}

func newMetadataProcessor(computeDiskClient daisyCompute.Client, args ImportArguments) processor {
	return &metadataProcessor{args, computeDiskClient}
}

func (p *metadataProcessor) process(pd persistentDisk,
	loggableBuilder *service.SingleImageImportLoggableBuilder) (persistentDisk, error) {

	// If this is not a UEFI disk, don't add "UEFI_COMPATIBLE" for it.
	if !pd.isUEFICompatible {
		return pd, nil
	}

	// If "UEFI_COMPATIBLE" has already existed on the disk, nothing extra needs to be done.
	diskName := daisyUtils.GetResourceID(pd.uri)
	d, err := p.computeDiskClient.GetDisk(p.args.Project, p.args.Zone, diskName)
	if err != nil {
		return pd, daisy.Errf("Failed to get disk: %v", err)
	}
	for _, f := range d.GuestOsFeatures {
		if f.Type == "UEFI_COMPATIBLE" {
			return pd, nil
		}
	}

	// Now let's add "UEFI_COMPATIBLE" to the disk's guestOsFeatures.
	// GuestOSFeatures are immutable properties. Therefore:
	// 1. Copy the existing disk, adding "UEFI_COMPATIBLE"
	// 2. Update the reference
	// 3. Delete the previous disk.
	newDiskName := fmt.Sprintf("%v-uefi", diskName)
	err = p.computeDiskClient.CreateDisk(p.args.Project, p.args.Zone, &compute.Disk{
		Name:            newDiskName,
		SourceDisk:      pd.uri,
		GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
	})
	if err != nil {
		return pd, daisy.Errf("Failed to create UEFI disk: %v", err)
	}
	log.Println("UEFI disk created: ", newDiskName)

	// Delete the old disk after the new disk is created.
	deleteDisk(p.computeDiskClient, p.args.Project, p.args.Zone, pd)

	// Update the new disk URI
	pd.uri = fmt.Sprintf("zones/%v/disks/%v", p.args.Zone, newDiskName)
	return pd, nil
}

func (p *metadataProcessor) cancel(reason string) bool {
	// Cancel is not performed since there is only one critical API call - CreateDisk
	return false
}

func (p *metadataProcessor) traceLogs() []string {
	return []string{}
}
