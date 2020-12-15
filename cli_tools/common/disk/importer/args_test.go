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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
)

func TestRequireImageName(t *testing.T) {
	assert.EqualError(t, expectFailedValidation(t, "-client_id=pantheon"), "-image_name has to be specified")
}

func TestTrimAndLowerImageName(t *testing.T) {
	assert.Equal(t, "gcp-is-great", parseAndValidate(t, "-image_name", "  GCP-is-GREAT  ").ImageName)
}

func TestTrimFamily(t *testing.T) {
	assert.Equal(t, "Ubuntu", parseAndValidate(t, "-family", "  Ubuntu  ").Family)
}

func TestTrimDescription(t *testing.T) {
	assert.Equal(t, "Ubuntu", parseAndValidate(t, "-description", "  Ubuntu  ").Description)
}

func TestParseLabelsToMap(t *testing.T) {
	expected := map[string]string{"internal": "true", "private": "false"}
	assert.Equal(t, expected, parseAndValidate(t, "-labels=internal=true,private=false").Labels)
}

func TestFailOnLabelSyntaxError(t *testing.T) {
	assert.Contains(t, expectFailedParse(t, "-labels=internal:true").Error(),
		"invalid value \"internal:true\" for flag -labels")
}

func TestPopulateStorageLocationIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:            "us-west2-a",
		region:          "us-west2",
		storageLocation: "us",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us", actual.StorageLocation)
}

func TestTrimAndLowerStorageLocation(t *testing.T) {
	assert.Equal(t, "eu", parseAndValidate(t, "-storage_location", "  EU  ").StorageLocation)
}

func TestPopulateWorkflowDir(t *testing.T) {
	assert.Regexp(t, ".*/daisy_workflows", parseAndValidate(t).WorkflowDir)
}

func TestFailWhenClientIdMissing(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t).Error(), "-client_id has to be specified")
}

func TestTrimAndLowerClientId(t *testing.T) {
	assert.Equal(t, "pantheon", parseAndValidate(t, "-client_id", " Pantheon ").ClientID)
}

func TestTrimClientVersion(t *testing.T) {
	assert.Equal(t, "301.0.0B", parseAndValidate(t, "-client_version", " 301.0.0B ").ClientVersion)
}

func TestTrimProject(t *testing.T) {
	assert.Equal(t, "TestProject", parseAndValidate(t, "-project", " TestProject ").Project)
}

func TestPopulateProjectIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:    "us-west2-a",
		region:  "us-west2",
		project: "the-project",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "the-project", actual.Project)
}

func TestTrimNetwork(t *testing.T) {
	assert.Equal(t, "global/networks/id", parseAndValidate(t, "-network", "  id  ").Network)
}

func TestTrimSubnet(t *testing.T) {
	assert.Equal(t, "regions/us-west2/subnetworks/sub-id", parseAndValidate(t, "-subnet", "  sub-id  ").Subnet)
}

func TestTrimAndLowerZone(t *testing.T) {
	assert.Equal(t, "us-central4-a", parseAndValidate(t, "-zone", "  us-central4-a  ").Zone)
}

func TestPopulateZoneIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2-a", actual.Zone)
}

func TestPopulateRegion(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2", actual.Region)
}

func TestScratchBucketPath(t *testing.T) {
	started := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	id := "abc"
	scratchDir := "gce-image-import-2009-11-10T23:00:00Z-abc"
	var flagtests = []struct {
		name      string
		bucketArg string
		expected  string
	}{
		{"no path", "gs://bucket", "gs://bucket/" + scratchDir},
		{"empty path", "gs://bucket/", "gs://bucket/" + scratchDir},
		{"with path", "gs://bucket/path", "gs://bucket/path/" + scratchDir},
		{"trim, no path", "  gs://bucket  ", "gs://bucket/" + scratchDir},
		{"trim, empty path", "  gs://bucket/  ", "gs://bucket/" + scratchDir},
		{"trim, with path", "  gs://bucket/path  ", "gs://bucket/path/" + scratchDir},
		{"populate when missing", "", "gs://fallback-bucket/" + scratchDir},
	}
	for _, tt := range flagtests {
		t.Run(tt.name, func(t *testing.T) {
			args := parse(t, "-scratch_bucket_gcs_path", tt.bucketArg)
			args.Started = started
			args.ExecutionID = id
			err := args.ValidateAndPopulate(mockPopulator{
				zone:          "us-west2-a",
				region:        "us-west2",
				scratchBucket: "gs://fallback-bucket/",
			}, mockSourceFactory{})

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, args.ScratchBucketGcsPath)
		})
	}
}

func TestTrimOauth(t *testing.T) {
	assert.Equal(t, "file.json", parseAndValidate(t, "-oauth", "  file.json ").Oauth)
}

func TestTrimComputeEndpoint(t *testing.T) {
	assert.Equal(t, "http://endpoint",
		parseAndValidate(t, "-compute_endpoint_override", "  http://endpoint ").ComputeEndpoint)
}

func TestGcsLogsDisabled(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-disable_gcs_logging=false").GcsLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_gcs_logging=true").GcsLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_gcs_logging").GcsLogsDisabled)
}

func TestCloudLogsDisabled(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-disable_cloud_logging=false").CloudLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_cloud_logging=true").CloudLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_cloud_logging").CloudLogsDisabled)
}

func TestStdoutLogsDisabled(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-disable_stdout_logging=false").StdoutLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_stdout_logging=true").StdoutLogsDisabled)
	assert.True(t, parseAndValidate(t, "-disable_stdout_logging").StdoutLogsDisabled)
}

func TestNoExternalIp(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-no_external_ip=false").NoExternalIP)
	assert.True(t, parseAndValidate(t, "-no_external_ip=true").NoExternalIP)
	assert.True(t, parseAndValidate(t, "-no_external_ip").NoExternalIP)
}

func TestPopulateNetworkAndSubnet(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedNetwork string
		expectedSubnet  string
	}{
		{
			name:            "populate network as default when network and subnet empty",
			expectedNetwork: "global/networks/default",
		},
		{
			name:            "qualify network when specified",
			args:            []string{"-network", "custom-network"},
			expectedNetwork: "global/networks/custom-network",
		},
		{
			name:           "don't populate empty network when subnet is specified",
			args:           []string{"-subnet", "custom-subnet"},
			expectedSubnet: "regions/us-west2/subnetworks/custom-subnet",
		},
		{
			name:            "qualify network and subnet when both specified",
			args:            []string{"-subnet", "custom-subnet", "-network", "custom-network"},
			expectedNetwork: "global/networks/custom-network",
			expectedSubnet:  "regions/us-west2/subnetworks/custom-subnet",
		},
		{
			name: "keep pre-qualified URIs",
			args: []string{
				"-subnet", "regions/us-west2/subnetworks/pre-qual-subnet",
				"-network", "global/networks/pre-qual-network"},
			expectedNetwork: "global/networks/pre-qual-network",
			expectedSubnet:  "regions/us-west2/subnetworks/pre-qual-subnet",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parseAndValidate(t, tt.args...)
			assert.Equal(t, tt.expectedNetwork, actual.Network)
			assert.Equal(t, tt.expectedSubnet, actual.Subnet)
		})
	}
}

func TestTrimSourceFile(t *testing.T) {
	assert.Equal(t, "gs://bucket/image.vmdk", parseAndValidate(
		t, "-source_file", " gs://bucket/image.vmdk ").SourceFile)
}

func TestTrimSourceImage(t *testing.T) {
	assert.Equal(t, "path/source-image", parseAndValidate(
		t, "-source_image", "  path/source-image  ").SourceImage)
}

func TestSourceObjectFromSourceImage(t *testing.T) {
	args := []string{"-source_image", "path/source-image", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gs://custom-bucket/",
	}, mockSourceFactory{
		expectedImage: "path/source-image",
		t:             t,
	})
	assert.NoError(t, err)
	assert.Equal(t, "path/source-image", actual.SourceImage)
	assert.Equal(t, "path/source-image", actual.Source.Path())
}

func TestSourceObjectFromSourceFile(t *testing.T) {
	args := []string{"-source_file", "gs://path/file", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gs://custom-bucket/",
	}, mockSourceFactory{
		expectedFile: "gs://path/file",
		t:            t,
	})
	assert.NoError(t, err)
	assert.Equal(t, "gs://path/file", actual.SourceFile)
	assert.Equal(t, "gs://path/file", actual.Source.Path())
}

func TestErrorWhenSourceValidationFails(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gs://custom-bucket/",
	}, mockSourceFactory{
		t:   t,
		err: errors.New("bad source"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad source")
}

func TestDataDiskSettable(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-data_disk=false", "-os=ubuntu-1804").DataDisk)
	assert.False(t, parseAndValidate(t, "-os=ubuntu-1804").DataDisk)
	assert.True(t, parseAndValidate(t, "-data_disk=true").DataDisk)
	assert.True(t, parseAndValidate(t, "-data_disk").DataDisk)
}

func TestTrimAndLowerOS(t *testing.T) {
	assert.Equal(t, "ubuntu-1804", parseAndValidate(t, "-os", "  UBUNTU-1804 ").OS)
}

func TestFailWhenOSNotRegistered(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t,
		"-os=android", "-client_id=c", "-image_name=i").Error(),
		"os `android` is invalid. Allowed values:")
}

func TestNoGuestEnvironmentSettable(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-data_disk=false", "-os=ubuntu-1804").DataDisk)
	assert.False(t, parseAndValidate(t, "-os=ubuntu-1804").DataDisk)
	assert.True(t, parseAndValidate(t, "-data_disk=true").DataDisk)
	assert.True(t, parseAndValidate(t, "-data_disk").DataDisk)
}

func TestBYOLDefaultsToFalse(t *testing.T) {
	assert.False(t, parseAndValidate(t).BYOL)
}

func TestBYOLIsSettable(t *testing.T) {
	assert.True(t, parseAndValidate(t, "-byol").BYOL)
}

func TestBYOLCanOnlyBeSpecifiedWhenDetectionEnabled(t *testing.T) {
	expectedError := "when -byol is specified, -data_disk, -os, and -custom_translate_workflow have to be empty"
	assert.Contains(t,
		expectFailedValidation(t, "-image_name=i", "-client_id=test", "-data_disk", "-byol").Error(),
		expectedError)
	assert.Contains(t,
		expectFailedValidation(t, "-image_name=i", "-client_id=test", "-os=ubuntu-1804", "-byol").Error(),
		expectedError)
	assert.Contains(t,
		expectFailedValidation(t, "-image_name=i", "-client_id=test", "-custom_translate_workflow=workflow.json", "-byol").Error(),
		expectedError)
}

func TestDurationHasDefaultValue(t *testing.T) {
	assert.Equal(t, time.Hour*2, parseAndValidate(t).Timeout)
}

func TestDurationIsSettable(t *testing.T) {
	assert.Equal(t, time.Hour*5, parseAndValidate(t, "-timeout=5h").Timeout)
}

func TestTrimCustomWorkflow(t *testing.T) {
	assert.Equal(t, "workflow.json", parseAndValidate(t,
		"-custom_translate_workflow", "  workflow.json  ").CustomWorkflow)
}

func TestValidateForConflictingArguments(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t,
		"-data_disk", "-os=ubuntu-1604", "-client_id=c", "-image_name=i").Error(),
		"when -data_disk is specified, -os and -custom_translate_workflow should be empty")

	assert.Contains(t, expectFailedValidation(t,
		"-data_disk", "-custom_translate_workflow=file.json", "-client_id=c", "-image_name=i").Error(),
		"when -data_disk is specified, -os and -custom_translate_workflow should be empty")

	assert.Contains(t, expectFailedValidation(t,
		"-os=ubuntu-1804", "-custom_translate_workflow=file.json", "-client_id=c", "-image_name=i").Error(),
		"-os and -custom_translate_workflow can't be both specified")
}

func TestUEFISettable(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-uefi_compatible=false").UefiCompatible)
	assert.True(t, parseAndValidate(t, "-uefi_compatible=true").UefiCompatible)
	assert.True(t, parseAndValidate(t, "-uefi_compatible").UefiCompatible)
}

func TestSysprepSettable(t *testing.T) {
	assert.False(t, parseAndValidate(t, "-sysprep_windows=false").SysprepWindows)
	assert.True(t, parseAndValidate(t, "-sysprep_windows=true").SysprepWindows)
	assert.True(t, parseAndValidate(t, "-sysprep_windows").SysprepWindows)
}

func TestImportArguments_DaisyAttrs(t *testing.T) {
	args := ImportArguments{
		Project:              "panda",
		Zone:                 "us-west",
		ScratchBucketGcsPath: "gs://bucket/path",
		Oauth:                "oauth-info",
		Timeout:              time.Hour * 3,
		ComputeEndpoint:      "endpoint-uri",
		GcsLogsDisabled:      true,
		CloudLogsDisabled:    true,
		StdoutLogsDisabled:   true,
		NoExternalIP:         true,
	}
	expected := daisycommon.WorkflowAttributes{
		Project:           "panda",
		Zone:              "us-west",
		GCSPath:           "gs://bucket/path",
		OAuth:             "oauth-info",
		Timeout:           "3h0m0s",
		ComputeEndpoint:   "endpoint-uri",
		DisableGCSLogs:    true,
		DisableCloudLogs:  true,
		DisableStdoutLogs: true,
		NoExternalIP:      true,
	}
	assert.Equal(t, expected, args.DaisyAttrs())
}

type mockPopulator struct {
	project         string
	zone            string
	region          string
	scratchBucket   string
	storageLocation string
	err             error
}

func (m mockPopulator) PopulateMissingParameters(project *string, client string, zone *string, region *string, scratchBucketGcsPath *string, file string, storageLocation *string) error {
	if m.err != nil {
		return m.err
	}
	if *project == "" {
		*project = m.project
	}
	if *zone == "" {
		*zone = m.zone
	}
	if *region == "" {
		*region = m.region
	}
	if *scratchBucketGcsPath == "" {
		*scratchBucketGcsPath = m.scratchBucket
	}
	if *storageLocation == "" {
		*storageLocation = m.storageLocation
	}
	return nil
}

type mockSource struct {
	sourcePath string
}

func (m mockSource) Path() string {
	return m.sourcePath
}

type mockSourceFactory struct {
	err                         error
	expectedFile, expectedImage string
	t                           *testing.T
}

func (m mockSourceFactory) Init(sourceFile, sourceImage string) (Source, error) {
	// Skip parameter verification unless they were provided when mock was setup.
	if m.expectedFile != "" {
		assert.Equal(m.t, m.expectedFile, sourceFile)
		return mockSource{sourcePath: sourceFile}, m.err

	}

	if m.expectedImage != "" {
		assert.Equal(m.t, m.expectedImage, sourceImage)
		return mockSource{sourcePath: sourceImage}, m.err
	}

	return mockSource{}, m.err
}

func parseAndValidate(t *testing.T, args ...string) ImportArguments {
	actual := parse(t, args...)
	err := actual.ValidateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})

	assert.NoError(t, err)
	return actual
}

func parse(t *testing.T, args ...string) ImportArguments {
	var hasClientID, hasImageName bool
	for _, arg := range args {
		if strings.HasPrefix(arg, "-client_id") {
			hasClientID = true
		} else if strings.HasPrefix(arg, "-image_name") {
			hasImageName = true
		}
	}

	if !hasClientID {
		args = append(args, "-client_id=pantheon")
	}

	if !hasImageName {
		args = append(args, "-image_name=name")
	}

	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	return actual
}

func expectFailedParse(t *testing.T, args ...string) error {
	_, err := NewImportArguments(args)
	assert.Error(t, err)
	return err
}

func expectFailedValidation(t *testing.T, args ...string) error {
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})

	assert.Error(t, err)
	return err
}
