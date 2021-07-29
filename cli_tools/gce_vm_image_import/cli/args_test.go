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

package cli

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
)

func Test_populateAndValidate_InitializesStarted(t *testing.T) {
	actual := parseAndPopulate(t, "-image_name=img")
	now := time.Now()
	if now.Sub(actual.Started) > time.Minute {
		t.Errorf("Expected Started to be initialized to current time. now=%q actual=%q", now, actual.Started)
	}
}

func Test_populateAndValidate_InitializesExecutionID(t *testing.T) {
	actual := parseAndPopulate(t, "-image_name=img")
	assert.NotEmpty(t, actual.ExecutionID)
}

func Test_populateAndValidate_SupportsCustomExecutionID(t *testing.T) {
	expected := uuid.New().String()
	actual := addRequiredArgsAndParse(t, "-execution_id="+expected)
	assert.Equal(t, expected, actual.ExecutionID)
}

func Test_populateAndValidate_TrimsAndLowerImageName(t *testing.T) {
	assert.Equal(t, "gcp-is-great", parseAndPopulate(t, "-image_name", "  GCP-is-GREAT  ").ImageName)
}

func Test_populateAndValidate_TrimsFamily(t *testing.T) {
	assert.Equal(t, "Ubuntu", parseAndPopulate(t, "-family", "  Ubuntu  ").Family)
}

func Test_populateAndValidate_TrimsDescription(t *testing.T) {
	assert.Equal(t, "Ubuntu", parseAndPopulate(t, "-description", "  Ubuntu  ").Description)
}

func Test_populateAndValidate_CreatesMapOfLabels(t *testing.T) {
	expected := map[string]string{"internal": "true", "private": "false"}
	assert.Equal(t, expected, parseAndPopulate(t, "-labels=internal=true,private=false").Labels)
}

func Test_populateAndValidate_FailsWhenLabelsHaveSyntaxError(t *testing.T) {
	assert.Contains(t, expectFailedParse(t, "-labels=internal:true").Error(),
		"invalid value \"internal:true\" for flag -labels")
}

func Test_populateAndValidate_TrimsAndLowerStorageLocation(t *testing.T) {
	assert.Equal(t, "eu", parseAndPopulate(t, "-storage_location", "  EU  ").StorageLocation)
}

func Test_populateAndValidate_TrimsAndLowerClientId(t *testing.T) {
	assert.Equal(t, "pantheon", parseAndPopulate(t, "-client_id", " Pantheon ").ClientID)
}

func Test_populateAndValidate_TrimsClientVersion(t *testing.T) {
	assert.Equal(t, "301.0.0B", parseAndPopulate(t, "-client_version", " 301.0.0B ").ClientVersion)
}

func Test_populateAndValidate_TrimsProject(t *testing.T) {
	assert.Equal(t, "TestProject", parseAndPopulate(t, "-project", " TestProject ").Project)
}

func Test_populateAndValidate_TrimsNetwork(t *testing.T) {
	assert.Equal(t, "id", parseAndPopulate(t, "-network", "  id  ").Network)
}

func Test_populateAndValidate_TrimsSubnet(t *testing.T) {
	assert.Equal(t, "sub-id", parseAndPopulate(t, "-subnet", "  sub-id  ").Subnet)
}

func Test_populateAndValidate_TrimsAndLowerZone(t *testing.T) {
	assert.Equal(t, "us-central4-a", parseAndPopulate(t, "-zone", "  us-central4-a  ").Zone)
}

func Test_populateAndValidate_TrimsOauth(t *testing.T) {
	assert.Equal(t, "file.json", parseAndPopulate(t, "-oauth", "  file.json ").Oauth)
}

func Test_populateAndValidate_TrimsComputeEndpoint(t *testing.T) {
	assert.Equal(t, "http://endpoint",
		parseAndPopulate(t, "-compute_endpoint_override", "  http://endpoint ").ComputeEndpoint)
}

func Test_populateAndValidate_TrimsComputeServiceAccount(t *testing.T) {
	assert.Equal(t, "",
		parseAndPopulate(t, "-compute_service_account", " 	").ComputeServiceAccount)
	assert.Equal(t, "email",
		parseAndPopulate(t, "-compute_service_account", " email	").ComputeServiceAccount)
}

func Test_populateAndValidate_SupportsGcsLogsDisabled(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-disable_gcs_logging=false").GcsLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_gcs_logging=true").GcsLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_gcs_logging").GcsLogsDisabled)
}

func Test_populateAndValidate_SupportsCloudLogsDisabled(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-disable_cloud_logging=false").CloudLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_cloud_logging=true").CloudLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_cloud_logging").CloudLogsDisabled)
}

func Test_populateAndValidate_SupportsStdoutLogsDisabled(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-disable_stdout_logging=false").StdoutLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_stdout_logging=true").StdoutLogsDisabled)
	assert.True(t, parseAndPopulate(t, "-disable_stdout_logging").StdoutLogsDisabled)
}

func Test_populateAndValidate_SupportsNoExternalIp(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-no_external_ip=false").NoExternalIP)
	assert.True(t, parseAndPopulate(t, "-no_external_ip=true").NoExternalIP)
	assert.True(t, parseAndPopulate(t, "-no_external_ip").NoExternalIP)
}

func Test_populateAndValidate_TrimsSourceFile(t *testing.T) {
	assert.Equal(t, "gs://bucket/image.vmdk", parseAndPopulate(
		t, "-source_file", " gs://bucket/image.vmdk ").SourceFile)
}

func Test_populateAndValidate_TrimsSourceImage(t *testing.T) {
	assert.Equal(t, "path/source-image", parseAndPopulate(
		t, "-source_image", "  path/source-image  ").SourceImage)
}

func Test_populateAndValidate_SupportsNoGuestEnvironment(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-no_guest_environment=false", "-os=ubuntu-1804").NoGuestEnvironment)
	assert.False(t, parseAndPopulate(t, "-os=ubuntu-1804").NoGuestEnvironment)
	assert.True(t, parseAndPopulate(t, "-no_guest_environment=true").NoGuestEnvironment)
	assert.True(t, parseAndPopulate(t, "-no_guest_environment").NoGuestEnvironment)
}

func Test_populateAndValidate_TrimsAndLowerOS(t *testing.T) {
	assert.Equal(t, "ubuntu-1804", parseAndPopulate(t, "-os", "  UBUNTU-1804 ").OS)
}

func Test_populateAndValidate_SupportsDataDisk(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-data_disk=false", "-os=ubuntu-1804").DataDisk)
	assert.False(t, parseAndPopulate(t, "-os=ubuntu-1804").DataDisk)
	assert.True(t, parseAndPopulate(t, "-data_disk=true").DataDisk)
	assert.True(t, parseAndPopulate(t, "-data_disk").DataDisk)
}

func Test_populateAndValidate_DefaultsBYOLToFalse(t *testing.T) {
	assert.False(t, parseAndPopulate(t).BYOL)
}

func Test_populateAndValidate_SupportsBYOL(t *testing.T) {
	assert.True(t, parseAndPopulate(t, "-byol").BYOL)
}

func Test_populateAndValidate_simplifyBYOLInputs(t *testing.T) {
	// The import module rejects requests when both BYOL and osID are specified.
	// This test ensures we simplify the user's request to follow this requirement.
	type test struct {
		args         []string
		expectedOSID string
	}
	for _, tc := range []test{
		{
			args:         []string{"-os=rhel-8", "-byol"},
			expectedOSID: "rhel-8-byol",
		}, {
			args:         []string{"-os=rhel-8-byol", "-byol"},
			expectedOSID: "rhel-8-byol",
		}, {
			args:         []string{"-os=xyz", "-byol"},
			expectedOSID: "xyz-byol",
		},
	} {
		t.Run(fmt.Sprintf("%+v", tc), func(t *testing.T) {
			result := parseAndPopulate(t, tc.args...)
			assert.Equal(t, tc.expectedOSID, result.OS)
			assert.False(t, result.BYOL, "Only enable BYOL when osID is empty, which causes OS detection")
		})
	}
}

func Test_populateAndValidate_TimeoutHasDefaultValue(t *testing.T) {
	assert.Equal(t, time.Hour*2, parseAndPopulate(t).Timeout)
}

func Test_populateAndValidate_SupportsTimeout(t *testing.T) {
	assert.Equal(t, time.Hour*5, parseAndPopulate(t, "-timeout=5h").Timeout)
}

func Test_populateAndValidate_TrimsCustomWorkflow(t *testing.T) {
	assert.Equal(t, "workflow.json", parseAndPopulate(t,
		"-custom_translate_workflow", "  workflow.json  ").CustomWorkflow)
}

func Test_populateAndValidate_SupportsUEFI(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-uefi_compatible=false").UefiCompatible)
	assert.True(t, parseAndPopulate(t, "-uefi_compatible=true").UefiCompatible)
	assert.True(t, parseAndPopulate(t, "-uefi_compatible").UefiCompatible)
}

func Test_populateAndValidate_SupportsSysprep(t *testing.T) {
	assert.False(t, parseAndPopulate(t, "-sysprep_windows=false").SysprepWindows)
	assert.True(t, parseAndPopulate(t, "-sysprep_windows=true").SysprepWindows)
	assert.True(t, parseAndPopulate(t, "-sysprep_windows").SysprepWindows)
}

func Test_populateAndValidate_ClientIdIsOptional(t *testing.T) {
	args, err := parseArgsFromUser([]string{"-image_name=i", "-data_disk"})
	assert.NoError(t, err)
	err = args.populateAndValidate(mockPopulator{
		zone:            "us-west2-a",
		region:          "us-west2",
		storageLocation: "us",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "", args.ClientID)
}

func Test_populateAndValidate_BackfillsStorageLocationIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
		zone:            "us-west2-a",
		region:          "us-west2",
		storageLocation: "us",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us", actual.StorageLocation)
}

func Test_populateAndValidate_BackfillsProjectIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
		zone:    "us-west2-a",
		region:  "us-west2",
		project: "the-project",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "the-project", actual.Project)
}

func Test_populateAndValidate_BackfillsZoneIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2-a", actual.Zone)
}

func Test_populateAndValidate_BackfillsRegionIfMissing(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})
	assert.NoError(t, err)
	assert.Equal(t, "us-west2", actual.Region)
}

func Test_populateAndValidate_UpdatesNetworkAndSubnet(t *testing.T) {
	actual := addRequiredArgsAndParse(t, "-network=", "-subnet=")
	err := actual.populateAndValidate(mockPopulator{
		zone:    "us-west2-a",
		region:  "us-west2",
		network: "fixed-network",
		subnet:  "fixed-subnet",
	}, mockSourceFactory{})

	assert.NoError(t, err)
	assert.Equal(t, "fixed-network", actual.Network)
	assert.Equal(t, "fixed-subnet", actual.Subnet)
}

func Test_populateAndValidate_CreatesSourceObjectFromSourceImage(t *testing.T) {
	args := []string{"-source_image", "path/source-image", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
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

func Test_populateAndValidate_CreatesSourceObjectFromSourceFile(t *testing.T) {
	args := []string{"-source_file", "gs://path/file", "-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
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

func Test_populateAndValidate_FailsWhenSourceValidateFails(t *testing.T) {
	args := []string{"-image_name=i", "-client_id=c", "-data_disk"}
	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	err = actual.populateAndValidate(mockPopulator{
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

func Test_populateAndValidate_StandardizesScratchBucketPath(t *testing.T) {
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
			args := addRequiredArgsAndParse(t, "-scratch_bucket_gcs_path", tt.bucketArg)
			args.Started = started
			args.ExecutionID = id
			err := args.populateAndValidate(mockPopulator{
				zone:          "us-west2-a",
				region:        "us-west2",
				scratchBucket: "gs://fallback-bucket/",
			}, mockSourceFactory{})

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, args.ScratchBucketGcsPath)
		})
	}
}

// fields here will override what's passed to PopulateMissingParameters
type mockPopulator struct {
	project         string
	zone            string
	region          string
	scratchBucket   string
	storageLocation string
	network         string
	subnet          string
	err             error
}

func (m mockPopulator) PopulateMissingParameters(project *string, client string, zone *string, region *string,
	scratchBucketGcsPath *string, file string, storageLocation, network, subnet *string) error {
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
	if *network == "" {
		*network = m.network
	}
	if *subnet == "" {
		*subnet = m.subnet
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

func (m mockSourceFactory) Init(sourceFile, sourceImage string) (importer.Source, error) {
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

func parseAndPopulate(t *testing.T, args ...string) imageImportArgs {
	actual := addRequiredArgsAndParse(t, args...)
	err := actual.populateAndValidate(mockPopulator{
		zone:   "us-west2-a",
		region: "us-west2",
	}, mockSourceFactory{})

	assert.NoError(t, err)
	return actual
}

func addRequiredArgsAndParse(t *testing.T, args ...string) imageImportArgs {
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

	actual, err := parseArgsFromUser(args)
	assert.NoError(t, err)
	return actual
}

func expectFailedParse(t *testing.T, args ...string) error {
	_, err := parseArgsFromUser(args)
	assert.Error(t, err)
	return err
}
