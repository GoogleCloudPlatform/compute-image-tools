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
)

func TestRequireImageName(t *testing.T) {
	assert.EqualError(t, expectFailedValidation(t, "-client_id=pantheon"), "-image_name has to be specified")
}

func TestTrimAndLowerImageName(t *testing.T) {
	assert.Equal(t, "gcp-is-great", expectSuccessfulParse(t, "-image_name", "  GCP-is-GREAT  ").ImageName)
}

func TestTrimFamily(t *testing.T) {
	assert.Equal(t, "Ubuntu", expectSuccessfulParse(t, "-family", "  Ubuntu  ").Family)
}

func TestTrimDescription(t *testing.T) {
	assert.Equal(t, "Ubuntu", expectSuccessfulParse(t, "-description", "  Ubuntu  ").Description)
}

func TestParseLabelsToMap(t *testing.T) {
	expected := map[string]string{"internal": "true", "private": "false"}
	assert.Equal(t, expected, expectSuccessfulParse(t, "-labels=internal=true,private=false").Labels)
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
	assert.Equal(t, "eu", expectSuccessfulParse(t, "-storage_location", "  EU  ").StorageLocation)
}

func TestPopulateCurrentDirectory(t *testing.T) {
	assert.NotEmpty(t, expectSuccessfulParse(t).CurrentExecutablePath)
}

func TestFailWhenClientIdMissing(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t).Error(), "-client_id has to be specified")
}

func TestTrimAndLowerClientId(t *testing.T) {
	assert.Equal(t, "pantheon", expectSuccessfulParse(t, "-client_id", " Pantheon ").ClientID)
}

func TestTrimProject(t *testing.T) {
	assert.Equal(t, "TestProject", expectSuccessfulParse(t, "-project", " TestProject ").Project)
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
	assert.Equal(t, "global/networks/id", expectSuccessfulParse(t, "-network", "  id  ").Network)
}

func TestTrimSubnet(t *testing.T) {
	assert.Equal(t, "regions/us-west2/subnetworks/sub-id", expectSuccessfulParse(t, "-subnet", "  sub-id  ").Subnet)
}

func TestTrimAndLowerZone(t *testing.T) {
	assert.Equal(t, "us-central4-a", expectSuccessfulParse(t, "-zone", "  us-central4-a  ").Zone)
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

func TestTrimScratchBucket(t *testing.T) {
	assert.Equal(t, "gcs://bucket", expectSuccessfulParse(t, "-scratch_bucket_gcs_path", "  gcs://bucket  ").ScratchBucketGcsPath)
}

func TestPopulateScratchBucketIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gcs://custom-bucket/",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "gcs://custom-bucket/", actual.ScratchBucketGcsPath)
}

func TestTrimOauth(t *testing.T) {
	assert.Equal(t, "file.json", expectSuccessfulParse(t, "-oauth", "  file.json ").Oauth)
}

func TestTrimComputeEndpoint(t *testing.T) {
	assert.Equal(t, "http://endpoint", expectSuccessfulParse(t, "-compute_endpoint_override", "  http://endpoint ").ComputeEndpoint)
}

func TestGcsLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_gcs_logging=false").GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging=true").GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging").GcsLogsDisabled)
}

func TestCloudLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_cloud_logging=false").CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging=true").CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging").CloudLogsDisabled)
}

func TestStdoutLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_stdout_logging=false").StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging=true").StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging").StdoutLogsDisabled)
}

func TestNoExternalIp(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-no_external_ip=false").NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip=true").NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip").NoExternalIP)
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
			actual := expectSuccessfulParse(t, tt.args...)
			assert.Equal(t, tt.expectedNetwork, actual.Network)
			assert.Equal(t, tt.expectedSubnet, actual.Subnet)
		})
	}
}

func TestTrimSourceFile(t *testing.T) {
	assert.Equal(t, "gcs://bucket/image.vmdk", expectSuccessfulParse(
		t, "-source_file", " gcs://bucket/image.vmdk ").SourceFile)
}

func TestTrimSourceImage(t *testing.T) {
	assert.Equal(t, "path/source-image", expectSuccessfulParse(
		t, "-source_image", "  path/source-image  ").SourceImage)
}

func TestSourceObjectFromSourceImage(t *testing.T) {
	args := []string{"-source_image", "path/source-image", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gcs://custom-bucket/",
	}, mockSourceFactory{
		expectedImage: "path/source-image",
		t:             t,
	})
	assert.NoError(t, err)
	assert.Equal(t, "path/source-image", actual.SourceImage)
	assert.Equal(t, "path/source-image", actual.Source.Path())
}

func TestSourceObjectFromSourceFile(t *testing.T) {
	args := []string{"-source_file", "gcs://path/file", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gcs://custom-bucket/",
	}, mockSourceFactory{
		expectedFile: "gcs://path/file",
		t:            t,
	})
	assert.NoError(t, err)
	assert.Equal(t, "gcs://path/file", actual.SourceFile)
	assert.Equal(t, "gcs://path/file", actual.Source.Path())
}

func TestErrorWhenSourceValidationFails(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gcs://custom-bucket/",
	}, mockSourceFactory{
		t:   t,
		err: errors.New("bad source"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad source")
}

func TestDataDiskSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-data_disk=false", "-os=ubuntu-1804").DataDisk)
	assert.False(t, expectSuccessfulParse(t, "-os=ubuntu-1804").DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk=true").DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk").DataDisk)
}

func TestTrimAndLowerOS(t *testing.T) {
	assert.Equal(t, "ubuntu-1804", expectSuccessfulParse(t, "-os", "  UBUNTU-1804 ").OS)
}

func TestFailWhenOSNotRegistered(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t,
		"-os=android", "-client_id=c", "-image_name=i").Error(),
		"os `android` is invalid. Allowed values:")
}

func TestNoGuestEnvironmentSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-data_disk=false", "-os=ubuntu-1804").DataDisk)
	assert.False(t, expectSuccessfulParse(t, "-os=ubuntu-1804").DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk=true").DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk").DataDisk)
}

func TestRequireDataOSOrWorkflow(t *testing.T) {
	assert.Contains(t, expectFailedValidation(t, "-client_id=c", "-image_name=i").Error(),
		"-data_disk, -os, or -custom_translate_workflow has to be specified")
}

func TestDurationHasDefaultValue(t *testing.T) {
	assert.Equal(t, time.Hour*2, expectSuccessfulParse(t).Timeout)
}

func TestDurationIsSettable(t *testing.T) {
	assert.Equal(t, time.Hour*5, expectSuccessfulParse(t, "-timeout=5h").Timeout)
}

func TestTrimCustomWorkflow(t *testing.T) {
	assert.Equal(t, "workflow.json", expectSuccessfulParse(t,
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
	assert.False(t, expectSuccessfulParse(t, "-uefi_compatible=false").UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible=true").UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible").UefiCompatible)
}

func TestSysprepSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-sysprep_windows=false").SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows=true").SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows").SysprepWindows)
}

type mockPopulator struct {
	project         string
	zone            string
	region          string
	scratchBucket   string
	storageLocation string
	err             error
}

func (m mockPopulator) PopulateMissingParameters(project *string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, storageLocation *string) error {
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

func expectSuccessfulParse(t *testing.T, args ...string) ImportArguments {
	var hasClientID, hasImageName, hasTranslationType bool
	for _, arg := range args {
		if strings.HasPrefix(arg, "-client_id") {
			hasClientID = true
		} else if strings.HasPrefix(arg, "-image_name") {
			hasImageName = true
		} else if strings.HasPrefix(arg, "-os") ||
			strings.HasPrefix(arg, "-data_disk") ||
			strings.HasPrefix(arg, "-custom_translate_workflow") {
			hasTranslationType = true
		}
	}

	if !hasClientID {
		args = append(args, "-client_id=pantheon")
	}

	if !hasImageName {
		args = append(args, "-image_name=name")
	}

	if !hasTranslationType {
		args = append(args, "-data_disk")
	}

	actual, err := NewImportArguments(args)
	assert.NoError(t, err)
	err = actual.ValidateAndPopulate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})

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
