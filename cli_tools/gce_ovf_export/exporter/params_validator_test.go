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

package ovfexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var validReleaseTracks = []string{"ga", "beta", "alpha"}

func TestInstanceNameAndMachineImageNameProvidedAtTheSameTime(t *testing.T) {
	params := getAllInstanceExportParams()
	params.MachineImageName = "machine-image-name"
	assertErrorOnValidate(t, params)
}

func TestInstanceExportFlagsInstanceNameNotProvided(t *testing.T) {
	params := getAllInstanceExportParams()
	params.InstanceName = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceExportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	params := getAllInstanceExportParams()
	params.DestinationURI = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceExportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	params := getAllInstanceExportParams()
	params.DestinationURI = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params)
}

func TestInstanceExportFlagsClientIdNotProvided(t *testing.T) {
	params := getAllInstanceExportParams()
	params.ClientID = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceExportFlagsAllValid(t *testing.T) {
	assert.Nil(t, ValidateAndParseParams(getAllInstanceExportParams(), validReleaseTracks))
}

func TestInstanceExportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	params := getAllInstanceExportParams()
	params.DestinationURI = "gs://bucket_name/"
	assert.Nil(t, ValidateAndParseParams(getAllInstanceExportParams(), validReleaseTracks))
}

func TestInstanceExportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	params := getAllInstanceExportParams()
	params.DestinationURI = "gs://bucket_name"
	assert.Nil(t, ValidateAndParseParams(getAllInstanceExportParams(), validReleaseTracks))
}

func TestInstanceExportFlagsInvalidOvfFormat(t *testing.T) {
	params := getAllInstanceExportParams()
	params.OvfFormat = "zip"
	assertErrorOnValidate(t, params)
}

func TestMachineImageExportFlagsAllValid(t *testing.T) {
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageExportParams(), validReleaseTracks))
}

func TestMachineImageExportFlagsMachineImageNameNotProvided(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.MachineImageName = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageExportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.DestinationURI = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageExportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.DestinationURI = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params)
}

func TestMachineImageExportFlagsClientIdNotProvided(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.ClientID = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageExportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.DestinationURI = "gs://bucket_name/"
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageExportParams(), validReleaseTracks))
}

func TestMachineImageExportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.DestinationURI = "gs://bucket_name"
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageExportParams(), validReleaseTracks))
}

func assertErrorOnValidate(t *testing.T, params *OVFExportParams) {
	assert.NotNil(t, ValidateAndParseParams(params, validReleaseTracks))
}

func TestMachineImageExportFlagsInvalidOvfFormat(t *testing.T) {
	params := getAllMachineImageExportParams()
	params.OvfFormat = "zip"
	assertErrorOnValidate(t, params)
}

func getAllInstanceExportParams() *OVFExportParams {
	project := "aProject"
	return &OVFExportParams{
		InstanceName:         "instance1",
		ClientID:             "aClient",
		DestinationURI:       "gs://ovfbucket/ovfpath/vmware.ova",
		Network:              "aNetwork",
		Subnet:               "aSubnet",
		Zone:                 "us-central1-c",
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
		BuildID:              "abc123",
		DiskExportFormat:     "vmdk",
	}
}

func getAllMachineImageExportParams() *OVFExportParams {
	project := "aProject"
	return &OVFExportParams{
		MachineImageName:     "machine-image1",
		ClientID:             "aClient",
		DestinationURI:       "gs://ovfbucket/ovfpath/vmware.ova",
		Network:              "aNetwork",
		Subnet:               "aSubnet",
		Zone:                 "us-central1-c",
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
		BuildID:              "abc123",
		DiskExportFormat:     "vmdk",
		OvfFormat:            "ovf",
	}
}
