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

package ovfimportparams

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceNameAndMachineImageNameProvidedAtTheSameTime(t *testing.T) {
	params := getAllInstanceImportParams()
	params.MachineImageName = "machine-image-name"
	assertErrorOnValidate(t, params)
}

func TestMachineImageStorageLocationProvidedForInstanceImport(t *testing.T) {
	params := getAllInstanceImportParams()
	params.MachineImageStorageLocation = "us-west2"
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsInstanceNameNotProvided(t *testing.T) {
	params := getAllInstanceImportParams()
	params.InstanceNames = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OvfOvaGcsPath = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OvfOvaGcsPath = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsClientIdNotProvided(t *testing.T) {
	params := getAllInstanceImportParams()
	params.ClientID = ""
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsLabelsInvalid(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Labels = "NOT_VALID_LABEL_DEFINITION"
	assertErrorOnValidate(t, params)
}

func TestInstanceImportFlagsAllValid(t *testing.T) {
	assert.Nil(t, ValidateAndParseParams(getAllInstanceImportParams()))
}

func TestInstanceImportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OvfOvaGcsPath = "gs://bucket_name/"
	assert.Nil(t, ValidateAndParseParams(getAllInstanceImportParams()))
}

func TestInstanceImportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	params := getAllInstanceImportParams()
	params.OvfOvaGcsPath = "gs://bucket_name"
	assert.Nil(t, ValidateAndParseParams(getAllInstanceImportParams()))
}

func TestInstanceImportHostnameInvalid(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Hostname = "an_invalid_host_name.an_invalid_domain"
	assertErrorOnValidate(t, params)
}

func TestInstanceImportHostnameTooLong(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Hostname = "a-host.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain"
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsAllValid(t *testing.T) {
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageImportParams()))
}

func TestMachineImageImportFlagsMachineImageNameNotProvided(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.MachineImageName = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.OvfOvaGcsPath = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.OvfOvaGcsPath = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsClientIdNotProvided(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.ClientID = ""
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsLabelsInvalid(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.Labels = "NOT_VALID_LABEL_DEFINITION"
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportFlagsAllValidBucketOnlyPathTrailingSlash(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.OvfOvaGcsPath = "gs://bucket_name/"
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageImportParams()))
}

func TestMachineImageImportFlagsAllValidBucketOnlyPathNoTrailingSlash(t *testing.T) {
	params := getAllMachineImageImportParams()
	params.OvfOvaGcsPath = "gs://bucket_name"
	assert.Nil(t, ValidateAndParseParams(getAllMachineImageImportParams()))
}

func TestMachineImageImportHostnameInvalid(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Hostname = "an_invalid_host_name.an_invalid_domain"
	assertErrorOnValidate(t, params)
}

func TestMachineImageImportHostnameTooLong(t *testing.T) {
	params := getAllInstanceImportParams()
	params.Hostname = "a-host.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain.a-domain"
	assertErrorOnValidate(t, params)
}

func assertErrorOnValidate(t *testing.T, params *OVFImportParams) {
	assert.NotNil(t, ValidateAndParseParams(params))
}

func getAllInstanceImportParams() *OVFImportParams {
	project := "aProject"
	return &OVFImportParams{
		InstanceNames:               "instance1",
		ClientID:                    "aClient",
		OvfOvaGcsPath:               "gs://ovfbucket/ovfpath/vmware.ova",
		NoGuestEnvironment:          true,
		CanIPForward:                true,
		DeletionProtection:          true,
		Description:                 "aDescription",
		Labels:                      "userkey1=uservalue1,userkey2=uservalue2",
		MachineType:                 "n1-standard-2",
		Network:                     "aNetwork",
		Subnet:                      "aSubnet",
		NetworkTier:                 "PREMIUM",
		PrivateNetworkIP:            "10.0.0.1",
		NoExternalIP:                true,
		NoRestartOnFailure:          true,
		OsID:                        "ubuntu-1404",
		ShieldedIntegrityMonitoring: true,
		ShieldedSecureBoot:          true,
		ShieldedVtpm:                true,
		Tags:                        "tag1=val1",
		Zone:                        "us-central1-c",
		BootDiskKmskey:              "aKey",
		BootDiskKmsKeyring:          "aKeyring",
		BootDiskKmsLocation:         "aKmsLocation",
		BootDiskKmsProject:          "aKmsProject",
		Timeout:                     "3h",
		Project:                     &project,
		ScratchBucketGcsPath:        "gs://bucket/folder",
		Oauth:                       "oAuthFilePath",
		Ce:                          "us-east1-c",
		GcsLogsDisabled:             true,
		CloudLogsDisabled:           true,
		StdoutLogsDisabled:          true,
		NodeAffinityLabelsFlag:      []string{"env,IN,prod,test"},
		Hostname:                    "a-host.a-domain",
	}
}

func getAllMachineImageImportParams() *OVFImportParams {
	project := "aProject"
	return &OVFImportParams{
		MachineImageName:            "machineImage1",
		MachineImageStorageLocation: "us-west2",
		ClientID:                    "aClient",
		OvfOvaGcsPath:               "gs://ovfbucket/ovfpath/vmware.ova",
		NoGuestEnvironment:          true,
		CanIPForward:                true,
		DeletionProtection:          true,
		Description:                 "aDescription",
		Labels:                      "userkey1=uservalue1,userkey2=uservalue2",
		MachineType:                 "n1-standard-2",
		Network:                     "aNetwork",
		Subnet:                      "aSubnet",
		NetworkTier:                 "PREMIUM",
		PrivateNetworkIP:            "10.0.0.1",
		NoExternalIP:                true,
		NoRestartOnFailure:          true,
		OsID:                        "ubuntu-1404",
		ShieldedIntegrityMonitoring: true,
		ShieldedSecureBoot:          true,
		ShieldedVtpm:                true,
		Tags:                        "tag1=val1",
		Zone:                        "us-central1-c",
		BootDiskKmskey:              "aKey",
		BootDiskKmsKeyring:          "aKeyring",
		BootDiskKmsLocation:         "aKmsLocation",
		BootDiskKmsProject:          "aKmsProject",
		Timeout:                     "3h",
		Project:                     &project,
		ScratchBucketGcsPath:        "gs://bucket/folder",
		Oauth:                       "oAuthFilePath",
		Ce:                          "us-east1-c",
		GcsLogsDisabled:             true,
		CloudLogsDisabled:           true,
		StdoutLogsDisabled:          true,
		NodeAffinityLabelsFlag:      []string{"env,IN,prod,test"},
		Hostname:                    "a-host.a-domain",
	}
}
