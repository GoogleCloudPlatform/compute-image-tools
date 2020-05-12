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

package args

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestImageSpec_RequireImageName(t *testing.T) {
	assert.EqualError(t, expectFailedParse(t, "-client_id=pantheon"), "-image_name has to be specified")
}

func TestImageSpec_TrimAndLowerImageName(t *testing.T) {
	assert.Equal(t, "gcp-is-great", expectSuccessfulParse(t, "-image_name", "  GCP-is-GREAT  ").Img.Name)
}

func TestImageSpec_TrimFamily(t *testing.T) {
	assert.Equal(t, "Ubuntu", expectSuccessfulParse(t, "-family", "  Ubuntu  ").Img.Family)
}

func TestImageSpec_TrimDescription(t *testing.T) {
	assert.Equal(t, "Ubuntu", expectSuccessfulParse(t, "-description", "  Ubuntu  ").Img.Description)
}

func TestImageSpec_ParseLabelsToMap(t *testing.T) {
	expected := map[string]string{"internal": "true", "private": "false"}
	assert.Equal(t, expected, expectSuccessfulParse(t, "-labels=internal=true,private=false").Img.Labels)
}

func TestImageSpec_FailOnLabelSyntaxError(t *testing.T) {
	assert.Contains(t, expectFailedParse(t, "-labels=internal:true").Error(),
		"invalid value \"internal:true\" for flag -labels")
}

func TestImageSpec_PopulateStorageLocationIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := ParseArgs(args, mockPopulator{
		zone:            "us-west2-a",
		region:          "us-west2",
		storageLocation: "us",
	})
	assert.NoError(t, err)
	assert.Equal(t, "us", actual.Img.StorageLocation)
}

func TestImageSpec_TrimAndLowerStorageLocation(t *testing.T) {
	assert.Equal(t, "eu", expectSuccessfulParse(t, "-storage_location", "  EU  ").Img.StorageLocation)
}

func TestEnvironment_PopulateCurrentDirectory(t *testing.T) {
	assert.NotEmpty(t, expectSuccessfulParse(t).Env.CurrentExecutablePath)
}

func TestEnvironment_FailWhenClientIdMissing(t *testing.T) {
	assert.Contains(t, expectFailedParse(t).Error(), "-client_id has to be specified")
}

func TestEnvironment_TrimAndLowerClientId(t *testing.T) {
	assert.Equal(t, "pantheon", expectSuccessfulParse(t, "-client_id", " Pantheon ").Env.ClientID)
}

func TestEnvironment_TrimProject(t *testing.T) {
	assert.Equal(t, "TestProject", expectSuccessfulParse(t, "-project", " TestProject ").Env.Project)
}

func TestImageSpec_PopulateProjectIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := ParseArgs(args, mockPopulator{
		zone:    "us-west2-a",
		region:  "us-west2",
		project: "the-project",
	})
	assert.NoError(t, err)
	assert.Equal(t, "the-project", actual.Env.Project)
}

func TestEnvironment_TrimNetwork(t *testing.T) {
	assert.Equal(t, "global/networks/id", expectSuccessfulParse(t, "-network", "  id  ").Env.Network)
}

func TestEnvironment_TrimSubnet(t *testing.T) {
	assert.Equal(t, "regions/us-west2/subnetworks/sub-id", expectSuccessfulParse(t, "-subnet", "  sub-id  ").Env.Subnet)
}

func TestEnvironment_TrimAndLowerZone(t *testing.T) {
	assert.Equal(t, "us-central4-a", expectSuccessfulParse(t, "-zone", "  us-central4-a  ").Env.Zone)
}

func TestImageSpec_PopulateZoneIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := ParseArgs(args, mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2-a", actual.Env.Zone)
}

func TestEnvironment_PopulateRegion(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := ParseArgs(args, mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2", actual.Env.Region)
}

func TestEnvironment_TrimScratchBucket(t *testing.T) {
	assert.Equal(t, "gcs://bucket", expectSuccessfulParse(t, "-scratch_bucket_gcs_path", "  gcs://bucket  ").Env.ScratchBucketGcsPath)
}

func TestImageSpec_PopulateScratchBucketIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := ParseArgs(args, mockPopulator{
		zone:          "us-west2-a",
		region:        "us-west2",
		scratchBucket: "gcs://custom-bucket/",
	})
	assert.NoError(t, err)
	assert.Equal(t, "gcs://custom-bucket/", actual.Env.ScratchBucketGcsPath)
}

func TestEnvironment_TrimOauth(t *testing.T) {
	assert.Equal(t, "file.json", expectSuccessfulParse(t, "-oauth", "  file.json ").Env.Oauth)
}

func TestEnvironment_TrimComputeEndpoint(t *testing.T) {
	assert.Equal(t, "http://endpoint", expectSuccessfulParse(t, "-compute_endpoint_override", "  http://endpoint ").Env.ComputeEndpoint)
}

func TestEnvironment_GcsLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_gcs_logging=false").Env.GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging=true").Env.GcsLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_gcs_logging").Env.GcsLogsDisabled)
}

func TestEnvironment_CloudLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_cloud_logging=false").Env.CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging=true").Env.CloudLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_cloud_logging").Env.CloudLogsDisabled)
}

func TestEnvironment_StdoutLogsDisabled(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-disable_stdout_logging=false").Env.StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging=true").Env.StdoutLogsDisabled)
	assert.True(t, expectSuccessfulParse(t, "-disable_stdout_logging").Env.StdoutLogsDisabled)
}

func TestEnvironment_NoExternalIp(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-no_external_ip=false").Env.NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip=true").Env.NoExternalIP)
	assert.True(t, expectSuccessfulParse(t, "-no_external_ip").Env.NoExternalIP)
}

func TestEnvironment_PopulateNetworkAndSubnet(t *testing.T) {

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
			assert.Equal(t, tt.expectedNetwork, actual.Env.Network)
			assert.Equal(t, tt.expectedSubnet, actual.Env.Subnet)
		})
	}
}

func TestTranslationSpec_TrimSourceFile(t *testing.T) {
	assert.Equal(t, "gcs://bucket/image.vmdk", expectSuccessfulParse(
		t, "-source_file", " gcs://bucket/image.vmdk ").Translation.SourceFile)
}

func TestTranslationSpec_TrimSourceImage(t *testing.T) {
	assert.Equal(t, "path/source-image", expectSuccessfulParse(
		t, "-source_image", "  path/source-image  ").Translation.SourceImage)
}

func TestTranslationSpec_DataDiskSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-data_disk=false", "-os=ubuntu-1804").Translation.DataDisk)
	assert.False(t, expectSuccessfulParse(t, "-os=ubuntu-1804").Translation.DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk=true").Translation.DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk").Translation.DataDisk)
}

func TestTranslationSpec_TrimAndLowerOS(t *testing.T) {
	assert.Equal(t, "ubuntu-1804", expectSuccessfulParse(t, "-os", "  UBUNTU-1804 ").Translation.OS)
}

func TestTranslationSpec_FailWhenOSNotRegistered(t *testing.T) {
	assert.Contains(t, expectFailedParse(t,
		"-os=android", "-client_id=c", "-image_name=i").Error(),
		"os `android` is invalid. Allowed values:")
}

func TestTranslationSpec_NoGuestEnvironmentSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-data_disk=false", "-os=ubuntu-1804").Translation.DataDisk)
	assert.False(t, expectSuccessfulParse(t, "-os=ubuntu-1804").Translation.DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk=true").Translation.DataDisk)
	assert.True(t, expectSuccessfulParse(t, "-data_disk").Translation.DataDisk)
}

func TestTranslationSpec_RequireDataOSOrWorkflow(t *testing.T) {
	assert.Contains(t, expectFailedParse(t, "-client_id=c", "-image_name=i").Error(),
		"-data_disk, -os, or -custom_translate_workflow has to be specified")
}

func TestTranslationSpec_DurationHasDefaultValue(t *testing.T) {
	assert.Equal(t, time.Hour*2, expectSuccessfulParse(t).Translation.Timeout)
}

func TestTranslationSpec_DurationIsSettable(t *testing.T) {
	assert.Equal(t, time.Hour*5, expectSuccessfulParse(t, "-timeout=5h").Translation.Timeout)
}

func TestTranslationSpec_TrimCustomWorkflow(t *testing.T) {
	assert.Equal(t, "workflow.json", expectSuccessfulParse(t,
		"-custom_translate_workflow", "  workflow.json  ").Translation.CustomWorkflow)
}

func TestTranslationSpec_ValidateForConflictingArguments(t *testing.T) {
	assert.Contains(t, expectFailedParse(t,
		"-data_disk", "-os=ubuntu-1604", "-client_id=c", "-image_name=i").Error(),
		"when -data_disk is specified, -os and -custom_translate_workflow should be empty")

	assert.Contains(t, expectFailedParse(t,
		"-data_disk", "-custom_translate_workflow=file.json", "-client_id=c", "-image_name=i").Error(),
		"when -data_disk is specified, -os and -custom_translate_workflow should be empty")

	assert.Contains(t, expectFailedParse(t,
		"-os=ubuntu-1804", "-custom_translate_workflow=file.json", "-client_id=c", "-image_name=i").Error(),
		"-os and -custom_translate_workflow can't be both specified")
}

func TestTranslationSpec_UEFISettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-uefi_compatible=false").Translation.UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible=true").Translation.UefiCompatible)
	assert.True(t, expectSuccessfulParse(t, "-uefi_compatible").Translation.UefiCompatible)
}

func TestTranslationSpec_SysprepSettable(t *testing.T) {
	assert.False(t, expectSuccessfulParse(t, "-sysprep_windows=false").Translation.SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows=true").Translation.SysprepWindows)
	assert.True(t, expectSuccessfulParse(t, "-sysprep_windows").Translation.SysprepWindows)
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

func expectSuccessfulParse(t *testing.T, args ...string) ParsedArguments {
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

	actual, err := ParseArgs(args, mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	})

	assert.NoError(t, err)
	return actual
}

func expectFailedParse(t *testing.T, args ...string) error {

	_, err := ParseArgs(args, mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	})

	assert.Error(t, err)
	return err
}
