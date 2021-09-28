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

package ovfexportdomain

import (
	"fmt"
	"time"
)

// TestProject is a test value for project flag
const TestProject = "a-project"

// TestZone is a test value for zone flag
const TestZone = "us-central1-c"

// TestRegion is a test value for region
const TestRegion = "us-central1"

// TestNetwork is a test value for network path
var TestNetwork = fmt.Sprintf("projects/%v/global/networks/%v", TestProject, "a-network")

// TestSubnet is a test value for subnet path
var TestSubnet = fmt.Sprintf("projects/%v/regions/%v/subnetworks/%v", TestProject, TestRegion, "a-subnet")

// GetAllInstanceExportArgs returns a new OVFExportArgs reference for instance export with default values
func GetAllInstanceExportArgs() *OVFExportArgs {
	var project = TestProject
	return &OVFExportArgs{
		InstanceName:         "instance1",
		ClientID:             "aClient",
		BuildID:              "aBuildID",
		DestinationURI:       "gs://ovfbucket/OVFpath/some-instance-ovf.ovf",
		DestinationDirectory: "gs://ovfbucket/OVFpath/",
		OvfName:              "ovfinst",
		Network:              TestNetwork,
		Subnet:               TestSubnet,
		Zone:                 TestZone,
		OsID:                 "ubuntu-1404",
		Timeout:              3 * time.Hour,
		Project:              project,
		ScratchBucketGcsPath: "gs://bucket/folder/",
		Oauth:                "oAuthFilePath",
		Ce:                   "us-east1-c",
		GcsLogsDisabled:      true,
		CloudLogsDisabled:    true,
		StdoutLogsDisabled:   true,
		ReleaseTrack:         GA,
		DiskExportFormat:     "vmdk",
		Region:               TestRegion,
	}
}

// GetAllMachineImageExportArgs returns a new OVFExportArgs reference for machine image export with default values
func GetAllMachineImageExportArgs() *OVFExportArgs {
	project := TestProject

	return &OVFExportArgs{
		MachineImageName:     "machine-image1",
		ClientID:             "aClient",
		DestinationURI:       "gs://ovfbucket/OVFpath",
		DestinationDirectory: "gs://ovfbucket/OVFpath/",
		OvfName:              "some-gmi-ovf",
		Network:              TestNetwork,
		Subnet:               TestSubnet,
		Zone:                 TestZone,
		OsID:                 "ubuntu-1404",
		Timeout:              3 * time.Hour,
		Project:              project,
		ScratchBucketGcsPath: "gs://bucket/folder",
		Oauth:                "oAuthFilePath",
		Ce:                   "us-east1-c",
		GcsLogsDisabled:      true,
		CloudLogsDisabled:    true,
		StdoutLogsDisabled:   true,
		ReleaseTrack:         GA,
		DiskExportFormat:     "vmdk",
		Region:               TestRegion,
	}
}
