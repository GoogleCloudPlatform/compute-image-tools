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

package main

import (
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	"github.com/stretchr/testify/assert"
)

func TestGetWorkflowPathWithoutFormatConversion(t *testing.T) {
	workflow := getWorkflowPath()
	expectedWorkflow := pathutils.ToWorkingDir(exportWorkflow, *currentExecutablePath)
	if workflow != expectedWorkflow {
		t.Errorf("%v != %v", workflow, expectedWorkflow)
	}
}

func TestGetWorkflowPathWithFormatConversion(t *testing.T) {
	defer testutils.SetStringP(&format, "vmdk")()
	workflow := getWorkflowPath()
	expectedWorkflow := pathutils.ToWorkingDir(exportAndConvertWorkflow, *currentExecutablePath)
	if workflow != expectedWorkflow {
		t.Errorf("%v != %v", workflow, expectedWorkflow)
	}
}

func TestFlagsSouceImageNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, sourceImageFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing source_image flag", t)
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, clientIDFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing client_id flag", t)
}

func TestFlagsDestinationUriNotProvided(t *testing.T) {
	defer testutils.BackupOsArgs()()
	cliArgs := getAllCliArgs()
	defer testutils.ClearStringFlag(cliArgs, destinationURIFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing destination_uri flag", t)
}

func buildOsArgsAndAssertErrorOnValidate(cliArgs map[string]interface{}, errorMsg string, t *testing.T) {
	testutils.BuildOsArgs(cliArgs)
	if err := validateAndParseFlags(); err == nil {
		t.Error(errorMsg)
	}
}

func TestBuildDaisyVarsWithoutFormatConversion(t *testing.T) {
	defer testutils.SetStringP(&region, "aRegion")()
	got := buildDaisyVars()

	assert.Equal(t, "global/images/anImage", got["source_image"])
	assert.Equal(t, "gs://bucket/exported_image", got["destination"])
	assert.Equal(t, "global/networks/aNetwork", got["export_network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", got["export_subnet"])
	assert.Equal(t, 4, len(got))
}

func TestBuildDaisyVarsWithFormatConversion(t *testing.T) {
	defer testutils.SetStringP(&region, "aRegion")()
	defer testutils.SetStringP(&format, "vmdk")()

	got := buildDaisyVars()

	assert.Equal(t, "global/images/anImage", got["source_image"])
	assert.Equal(t, "gs://bucket/exported_image", got["destination"])
	assert.Equal(t, "vmdk", got["format"])
	assert.Equal(t, "global/networks/aNetwork", got["export_network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", got["export_subnet"])
	assert.Equal(t, 5, len(got))
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{}{
		clientIDFlagKey:             "aClient",
		destinationURIFlagKey:       "gs://bucket/exported_image",
		sourceImageFlagKey:          "anImage",
		"format":                    "",
		"project":                   "aProject",
		"network":                   "aNetwork",
		"subnet":                    "aSubnet",
		"zone":                      "us-central1-c",
		"timeout":                   "2h",
		"scratch_bucket_gcs_path":   "gs://bucket/folder",
		"oauth":                     "oAuthFilePath",
		"compute_endpoint_override": "us-east1-c",
		"disable_gcs_logging":       true,
		"disable_cloud_logging":     true,
		"disable_stdout_logging":    true,
		"labels":                    "userkey1=uservalue1,userkey2=uservalue2",
	}
}
