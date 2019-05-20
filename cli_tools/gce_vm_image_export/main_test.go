//  Copyright 2018 Google Inc. All Rights Reserved.
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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"testing"

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

func TestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(t *testing.T) {
	defer testutils.SetStringP(&zone, "")()
	defer testutils.SetStringP(&scratchBucketGcsPath, "gs://scratchbucket/scratchpath")()
	defer testutils.SetStringP(&destinationURI, "gs://destbucket/destfile")()
	defer testutils.SetStringP(&project, "a_project")()

	err :=
			paramutils.RunTestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(
				t, zone, region, scratchBucketGcsPath, destinationURI, project, "scratchbucket",  "europe-north1", "europe-north1-b")

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-b", *zone)
	assert.Equal(t, "europe-north1", *region)
	assert.Equal(t, "gs://scratchbucket/scratchpath", *scratchBucketGcsPath)
}

func TestPopulateMissingParametersCreatesScratchBucketIfNotProvided(t *testing.T) {
	defer testutils.SetStringP(&zone, "")()
	defer testutils.SetStringP(&scratchBucketGcsPath, "")()
	defer testutils.SetStringP(&destinationURI, "gs://destbucket/destfile")()
	defer testutils.SetStringP(&project, "a_project")()

	err :=
			paramutils.RunTestPopulateMissingParametersCreatesScratchBucketIfNotProvided(
				t, zone, region, scratchBucketGcsPath, destinationURI, project, "a_project", "new_scratch_bucket", "europe-north1", "europe-north1-c")

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", *zone)
	assert.Equal(t, "europe-north1", *region)
	assert.Equal(t, "gs://new_scratch_bucket/", *scratchBucketGcsPath)
}

func TestPopulateProjectIfMissingProjectPopulatedFromGCE(t *testing.T) {
	defer testutils.SetStringP(&project, "")()

	err :=
			paramutils.RunTestPopulateProjectIfMissingProjectPopulatedFromGCE(t, project, "gce_project")

	assert.Nil(t, err)
	assert.Equal(t, "gce_project", *project)
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
