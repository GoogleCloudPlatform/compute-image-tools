//  Copyright 2021 Google Inc. All Rights Reserved.
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisyutils"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/validation"
)

func Test_FixBYOLAndOSFlags(t *testing.T) {
	for _, tt := range []struct {
		originalOSID, expectedOSID string
		originalBYOL, expectedBYOL bool
	}{{
		originalOSID: "rhel-8",
		originalBYOL: true,
		expectedOSID: "rhel-8-byol",
		expectedBYOL: false,
	}, {
		originalOSID: "rhel-8-byol",
		originalBYOL: true,
		expectedOSID: "rhel-8-byol",
		expectedBYOL: false,
	}, {
		originalOSID: "rhel-8",
		originalBYOL: false,
		expectedOSID: "rhel-8",
		expectedBYOL: false,
	}, {
		originalOSID: "rhel-8-byol",
		originalBYOL: false,
		expectedOSID: "rhel-8-byol",
		expectedBYOL: false,
	}, {
		originalOSID: "",
		originalBYOL: true,
		expectedOSID: "",
		expectedBYOL: true,
	}, {
		originalOSID: "",
		originalBYOL: false,
		expectedOSID: "",
		expectedBYOL: false,
	},
	} {
		t.Run(fmt.Sprintf("%+v", tt), func(t *testing.T) {
			FixBYOLAndOSArguments(&tt.originalOSID, &tt.originalBYOL)
			assert.Equal(t, tt.expectedOSID, tt.originalOSID)
			assert.Equal(t, tt.expectedBYOL, tt.originalBYOL)
		})
	}
}

func Test_validate_RequiresImageName(t *testing.T) {
	request := makeValidRequest()
	request.ImageName = ""
	assertMissingField(t, request, "image_name")
}

func Test_validate_ValidatesImageNameUsesValidationUtil(t *testing.T) {
	request := makeValidRequest()
	request.ImageName = "-no-starting-dashes"
	assert.Equal(t, request.validate(), validation.ValidateImageName(request.ImageName))
}

func Test_validate_AllowsValidImageName(t *testing.T) {
	// Allowable name format: https://cloud.google.com/compute/docs/reference/rest/v1/images
	for _, imgName := range []string{
		"dashes-allowed-inside",
		"a", // min length is 1
		"o-----equal-to-max-63-----------------------------------------o",
	} {
		t.Run(imgName, func(t *testing.T) {
			request := makeValidRequest()
			request.ImageName = imgName
			assert.NoError(t, request.validate())
		})
	}
}

func Test_validate_RequiresExecutionID(t *testing.T) {
	request := makeValidRequest()
	request.ExecutionID = ""
	assertMissingField(t, request, "execution_id")
}

func Test_validate_RequiresWorkflowDir(t *testing.T) {
	request := makeValidRequest()
	request.WorkflowDir = ""
	assertMissingField(t, request, "workflow_dir")
}

func Test_validate_NetworkIsOptional(t *testing.T) {
	request := makeValidRequest()
	request.Network = ""
	assert.NoError(t, request.validate())
}

func Test_validate_SubnetIsOptional(t *testing.T) {
	request := makeValidRequest()
	request.Subnet = ""
	assert.NoError(t, request.validate())
}

func Test_validate_RequiresProject(t *testing.T) {
	request := makeValidRequest()
	request.Project = ""
	assertMissingField(t, request, "project")
}

func Test_validate_RequiresScratchBucket(t *testing.T) {
	request := makeValidRequest()
	request.ScratchBucketGcsPath = ""
	assertMissingField(t, request, "scratch_bucket_gcs_path")
}

func Test_validate_RequiresSource(t *testing.T) {
	request := makeValidRequest()
	request.Source = nil
	assertMissingField(t, request, "source")
}

func Test_validate_RequiresTimeout(t *testing.T) {
	request := makeValidRequest()
	request.Timeout = 0
	assertMissingField(t, request, "timeout")
}

func Test_validate_RequiresZone(t *testing.T) {
	request := makeValidRequest()
	request.Zone = ""
	assertMissingField(t, request, "zone")
}

func Test_validate_ValidatesExecutionId(t *testing.T) {
	request := makeValidRequest()
	request.ExecutionID = ""
	assertMissingField(t, request, "execution_id")
}

func assertMissingField(t *testing.T, request ImageImportRequest, fieldName string) bool {
	err := request.validate()
	return assert.EqualError(t, err, fieldName+" has to be specified")
}

func makeValidRequest() ImageImportRequest {
	return ImageImportRequest{
		ExecutionID:           "execution-id",
		CloudLogsDisabled:     true,
		ComputeEndpoint:       "https://google.com",
		ComputeServiceAccount: "csa",
		WorkflowDir:           "path/to/workflows",
		DataDisk:              false,
		DaisyLogLinePrefix:    "import-image",
		Description:           "description",
		Family:                "family",
		GcsLogsDisabled:       true,
		ImageName:             "ubuntu20",
		Inspect:               true,
		Labels:                nil,
		Network:               "default",
		NoExternalIP:          true,
		NoGuestEnvironment:    true,
		Oauth:                 "encoded-oauth",
		OS:                    "ubuntu-2004",
		Project:               "project-name",
		ScratchBucketGcsPath:  "bucket-name-execution-id",
		Source:                fileSource{},
		StdoutLogsDisabled:    true,
		StorageLocation:       "us",
		Subnet:                "default",
		SysprepWindows:        true,
		Timeout:               time.Hour,
		UefiCompatible:        true,
		Zone:                  "us-central1-a",
	}
}

func Test_validate_FailsWhenOSNotRegistered(t *testing.T) {
	request := makeValidRequest()
	request.OS = "rare-distro-12"
	err := request.validate()
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.Regexp(t, "`rare-distro-12` is invalid", err.Error())
}

func Test_validate_ChecksForConflictingArguments(t *testing.T) {
	var flagtests = []struct {
		name          string
		request       ImageImportRequest
		expectedError string
	}{
		{
			request:       ImageImportRequest{DataDisk: true, OS: "ubuntu"},
			expectedError: "when -data_disk is specified, -os and -custom_translate_workflow should be empty",
		},
		{
			request:       ImageImportRequest{DataDisk: true, CustomWorkflow: "workflow.json"},
			expectedError: "when -data_disk is specified, -os and -custom_translate_workflow should be empty",
		},
		{
			request:       ImageImportRequest{OS: "ubuntu", CustomWorkflow: "workflow.json"},
			expectedError: "-os and -custom_translate_workflow can't be both specified",
		},
		{
			request:       ImageImportRequest{BYOL: true, OS: "ubuntu"},
			expectedError: "when -byol is specified, -data_disk, -os, and -custom_translate_workflow have to be empty",
		},
		{
			request:       ImageImportRequest{BYOL: true, CustomWorkflow: "workflow.json"},
			expectedError: "when -byol is specified, -data_disk, -os, and -custom_translate_workflow have to be empty",
		},
		{
			request:       ImageImportRequest{BYOL: true, DataDisk: true},
			expectedError: "when -byol is specified, -data_disk, -os, and -custom_translate_workflow have to be empty",
		},
	}
	for _, tt := range flagtests {
		t.Run(tt.name, func(t *testing.T) {
			toValidate := makeValidRequest()
			toValidate.DataDisk = tt.request.DataDisk
			toValidate.OS = tt.request.OS
			toValidate.CustomWorkflow = tt.request.CustomWorkflow
			toValidate.BYOL = tt.request.BYOL
			toValidate.DataDisk = tt.request.DataDisk
			err := toValidate.validate()
			assert.EqualError(t, err, tt.expectedError)
		})
	}
}

func Test_EnvironmentSettings(t *testing.T) {
	request := ImageImportRequest{
		Project:               "panda",
		Zone:                  "us-west",
		ScratchBucketGcsPath:  "gs://bucket/path",
		Oauth:                 "oauth-info",
		Timeout:               time.Hour * 3,
		ComputeEndpoint:       "endpoint-uri",
		DaisyLogLinePrefix:    "disk-0",
		GcsLogsDisabled:       true,
		CloudLogsDisabled:     true,
		StdoutLogsDisabled:    true,
		Network:               "network",
		Subnet:                "subnet",
		ComputeServiceAccount: "email@example.com",
		NoExternalIP:          true,
		WorkflowDir:           "workflow-dir",
	}
	expected := daisyutils.EnvironmentSettings{
		Project:               "panda",
		Zone:                  "us-west",
		GCSPath:               "gs://bucket/path",
		OAuth:                 "oauth-info",
		Timeout:               "3h0m0s",
		ComputeEndpoint:       "endpoint-uri",
		DaisyLogLinePrefix:    "disk-0",
		DisableGCSLogs:        true,
		DisableCloudLogs:      true,
		DisableStdoutLogs:     true,
		Network:               "network",
		Subnet:                "subnet",
		ComputeServiceAccount: "email@example.com",
		NoExternalIP:          true,
		WorkflowDirectory:     "workflow-dir",
	}
	assert.Equal(t, expected, request.EnvironmentSettings())
}
