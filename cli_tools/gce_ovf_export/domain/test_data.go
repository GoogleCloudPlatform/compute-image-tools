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

const TestProject = "aProject"
const TestZone = "us-central1-c"
const TestRegion = "us-central1"
const TestSubnet = "aSubnet"

func GetAllInstanceExportParams() *OVFExportParams {
	var project = TestProject
	return &OVFExportParams{
		InstanceName:         "instance1",
		ClientID:             "aClient",
		DestinationURI:       "gs://ovfbucket/OVFpath",
		Network:              "aNetwork",
		Subnet:               TestSubnet,
		Zone:                 TestZone,
		OsID:                 "ubuntu-1404",
		BootDiskKmskey:       "aKey",
		BootDiskKmsKeyring:   "aKeyring",
		BootDiskKmsLocation:  "aKmsLocation",
		BootDiskKmsProject:   "aKmsProject",
		Timeout:              "3h",
		Project:              &project,
		ScratchBucketGcsPath: "gs://bucket/folder",
		Oauth:                "oAuthFilePath",
		Ce:                   "us-east1-c",
		GcsLogsDisabled:      true,
		CloudLogsDisabled:    true,
		StdoutLogsDisabled:   true,
		ReleaseTrack:         "ga",
		DiskExportFormat:     "vmdk",
		Region:               TestRegion,
	}
}

func GetAllMachineImageExportParams() *OVFExportParams {
	project := TestProject

	return &OVFExportParams{
		MachineImageName:     "machine-image1",
		ClientID:             "aClient",
		DestinationURI:       "gs://ovfbucket/OVFpath",
		Network:              "aNetwork",
		Subnet:               "aSubnet",
		Zone:                 TestZone,
		OsID:                 "ubuntu-1404",
		BootDiskKmskey:       "aKey",
		BootDiskKmsKeyring:   "aKeyring",
		BootDiskKmsLocation:  "aKmsLocation",
		BootDiskKmsProject:   "aKmsProject",
		Timeout:              "3h",
		Project:              &project,
		ScratchBucketGcsPath: "gs://bucket/folder",
		Oauth:                "oAuthFilePath",
		Ce:                   "us-east1-c",
		GcsLogsDisabled:      true,
		CloudLogsDisabled:    true,
		StdoutLogsDisabled:   true,
		ReleaseTrack:         "ga",
		DiskExportFormat:     "vmdk",
		OvfFormat:            "ovf",
		Region:               TestRegion,
	}
}
