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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

type diskMutationProcessor struct {
	args              ImportArguments
	computeDiskClient daisyCompute.Client
}

func (p *diskMutationProcessor) process(pd persistentDisk) (persistentDisk, error) {
	// If UEFI_COMPATIBLE is enforced in user input args (b.uefiCompatible),
	// then it has been honored in inflation stage, so no need to recreate a new disk here.
	if p.args.UefiCompatible {
		return pd, nil
	}

	if !pd.isUEFIDetected {
		return pd, nil
	}

	// Due to GuestOS features limitations, a new disk needs to be created to add the additional "UEFI_COMPATIBLE"
	// and the old disk will be deleted.
	diskName := fmt.Sprintf("disk-%v-uefi", p.args.ExecutionID)
	err := p.computeDiskClient.CreateDisk(p.args.Project, p.args.Zone, &compute.Disk{
		Name:            diskName,
		SourceDisk:      pd.uri,
		GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
	})
	if err != nil {
		return pd, daisy.Errf("Failed to create UEFI disk: %v", err)
	}
	log.Println("UEFI disk created: ", diskName)

	// Cleanup the old disk after the new disk is created.
	cleanupDisk(p.computeDiskClient, p.args.Project, p.args.Zone, pd)

	// Update the new disk URI
	pd.uri = fmt.Sprintf("zones/%v/disks/%v", p.args.Zone, diskName)
	return pd, nil
}

func (p *diskMutationProcessor) cancel(reason string) bool {
	//indicate cancel was not performed
	return false
}

func (p *diskMutationProcessor) traceLogs() []string {
	return []string{}
}

func newDiskMutationProcessor(computeDiskClient daisyCompute.Client,
	args ImportArguments) processor {

	return &diskMutationProcessor{
		args:              args,
		computeDiskClient: computeDiskClient,
	}
}
