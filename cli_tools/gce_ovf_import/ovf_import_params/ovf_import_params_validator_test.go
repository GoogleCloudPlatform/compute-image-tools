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

func TestFlagsInstanceNameNotProvided(t *testing.T) {
	params := getAllParams()
	params.InstanceNames = ""
	assertErrorOnValidate(t, params)
}

func TestFlagsOvfGcsPathFlagKeyNotProvided(t *testing.T) {
	params := getAllParams()
	params.OvfOvaGcsPath = ""
	assertErrorOnValidate(t, params)
}

func TestFlagsOvfGcsPathFlagNotValid(t *testing.T) {
	params := getAllParams()
	params.OvfOvaGcsPath = "NOT_GCS_PATH"
	assertErrorOnValidate(t, params)
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	params := getAllParams()
	params.ClientID = ""
	assertErrorOnValidate(t, params)
}

func TestFlagsLabelsInvalid(t *testing.T) {
	params := getAllParams()
	params.Labels = "NOT_VALID_LABEL_DEFINITION"
	assertErrorOnValidate(t, params)
}

func TestFlagsOSAndNoTranslateBothSet(t *testing.T) {
	params := getAllParams()
	params.NoTranslate = true
	assertErrorOnValidate(t, params)
}

func TestFlagsAllValid(t *testing.T) {
	assert.Nil(t, ValidateAndParseParams(getAllParams()))
}

func assertErrorOnValidate(t *testing.T, params *OVFImportParams) {
	assert.NotNil(t, ValidateAndParseParams(params))
}

func getAllParams() *OVFImportParams {
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
	}
}
