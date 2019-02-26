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
	"cloud.google.com/go/storage"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"os"
	"testing"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe-north1-a", "europe-north1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		got, err := getRegion()
		if test.want != got {
			t.Errorf("%v != %v", test.want, got)
		} else if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		}
	}
	zone = oldZone
}

func TestPopulateRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	oldZone := zone
	for _, test := range tests {
		zone = &test.input
		region = nil
		err := populateRegion()
		if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		} else if region != nil && test.want != *region {
			t.Errorf("%v != %v", test.want, *region)
		}
	}
	zone = oldZone
}

func TestGetWorkflowPathsFromImage(t *testing.T) {
	defer setStringP(&sourceImage, "image-1")()
	defer setStringP(&osID, "ubuntu-1404")()
	workflow, translate := getWorkflowPaths()
	if workflow != importFromImageWorkflow && translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importFromImageWorkflow)
	}
}

func TestGetWorkflowPathsDataDisk(t *testing.T) {
	defer setBoolP(&dataDisk, true)()
	workflow, translate := getWorkflowPaths()
	if workflow != importWorkflow && translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, importWorkflow)
	}
}

func TestGetWorkflowPathsFromFile(t *testing.T) {
	homeDir := "/home/gce/"
	defer setBoolP(&dataDisk, false)()
	defer setStringP(&sourceImage, "image-1")()
	defer setStringP(&osID, "ubuntu-1404")()
	defer setStringP(&sourceImage, "")()
	defer setStringP(&currentExecutablePath, homeDir+"executable")()

	workflow, translate := getWorkflowPaths()

	if workflow != homeDir+importAndTranslateWorkflow {
		t.Errorf("resulting workflow path `%v` does not match expected `%v`", workflow, homeDir+importAndTranslateWorkflow)
	}

	if translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("resulting translate workflow path `%v` does not match expected `%v`", translate, "ubuntu/translate_ubuntu_1404.wf.json")
	}
}

func TestFlagsImageNameNotProvided(t *testing.T) {
	err := validateAndParseFlags()
	expected := fmt.Errorf("The flag -image_name must be provided")
	if err != expected && err.Error() != expected.Error() {
		t.Errorf("%v != %v", err, expected)
	}
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, clientIDFlagKey, &clientID)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing client_id flag", t)
}

func TestFlagsDataDiskOrOSFlagsNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "os", &osID)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing os or data_disk flag", t)
}

func TestFlagsDataDiskAndOSFlagsBothProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for both os and data_disk set at the same time", t)
}

func TestFlagsSourceFileOrSourceImageNotProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_file", &sourceFile)()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for missing source_file or source_image flag", t)
}

func TestFlagsSourceFileAndSourceImageBothProvided(t *testing.T) {
	defer backupOsArgs()()
	cliArgs := getAllCliArgs()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgsAndAssertErrorOnValidate(cliArgs, "Expected error for both source_file and source_image flags set", t)
}

func buildOsArgsAndAssertErrorOnValidate(cliArgs map[string]interface{}, errorMsg string, t *testing.T) {
	buildOsArgs(cliArgs)
	if err := validateAndParseFlags(); err == nil {
		t.Error(errorMsg)
	}
}

func TestFlagsSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidSourceFile(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	cliArgs["source_file"] = "invalidSourceFile"
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestFlagsSourceImage(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_file", &sourceFile)()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsDataDisk(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	defer clearStringFlag(cliArgs, "os", &osID)()
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidOS(t *testing.T) {
	defer backupOsArgs()()

	cliArgs := getAllCliArgs()
	defer clearBoolFlag(cliArgs, "data_disk", &dataDisk)()
	defer clearStringFlag(cliArgs, "source_image", &sourceImage)()
	cliArgs["os"] = "invalidOs"
	buildOsArgs(cliArgs)

	if err := validateAndParseFlags(); err == nil {
		t.Errorf("Expected error")
	}
}

func TestUpdateWorkflowInstancesConfiguredForNoExternalIP(t *testing.T) {
	defer setBoolP(&noExternalIP, true)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateAllInstanceNoExternalIP(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 0 {
		t.Errorf("Instance AccessConfigs not empty")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfExternalIPAllowed(t *testing.T) {
	defer setBoolP(&noExternalIP, false)()

	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	updateAllInstanceNoExternalIP(w)

	if len((*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces[0].AccessConfigs) != 1 {
		t.Errorf("Instance AccessConfigs doesn't have exactly one instance")
	}
}

func TestUpdateWorkflowInstancesNotModifiedIfNoNetworkInterfaceElement(t *testing.T) {
	defer setBoolP(&noExternalIP, true)()
	w := createWorkflowWithCreateInstanceNetworkAccessConfig()
	(*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces = nil
	updateAllInstanceNoExternalIP(w)

	if (*w.Steps["ci"].CreateInstances)[0].Instance.NetworkInterfaces != nil {
		t.Errorf("Instance NetworkInterfaces should stay nil if nil before update")
	}
}

func TestBuildDaisyVarsFromDisk(t *testing.T) {
	defer setStringP(&imageName, "image-a")()
	defer setBoolP(&noGuestEnvironment, true)()
	defer setStringP(&sourceFile, "source-file-path")()
	defer setStringP(&sourceImage, "")()
	defer setStringP(&family, "a-family")()
	defer setStringP(&description, "a-description")()
	defer setStringP(&network, "a-network")()
	defer setStringP(&subnet, "a-subnet")()
	defer setStringP(&region, "a-region")()

	got := buildDaisyVars("translate/workflow/path")

	assert.Equal(t, got["image_name"], "image-a")
	assert.Equal(t, got["translate_workflow"], "translate/workflow/path")
	assert.Equal(t, got["install_gce_packages"], "false")
	assert.Equal(t, got["source_disk_file"], "source-file-path")
	assert.Equal(t, got["family"], "a-family")
	assert.Equal(t, got["description"], "a-description")
	assert.Equal(t, got["import_network"], "global/networks/a-network")
	assert.Equal(t, got["import_subnet"], "regions/a-region/subnetworks/a-subnet")
	assert.Equal(t, len(got), 8)
}

func TestBuildDaisyVarsFromImage(t *testing.T) {
	defer setStringP(&imageName, "image-a")()
	defer setBoolP(&noGuestEnvironment, true)()
	defer setStringP(&sourceFile, "")()
	defer setStringP(&sourceImage, "source-image")()
	defer setStringP(&family, "a-family")()
	defer setStringP(&description, "a-description")()
	defer setStringP(&network, "a-network")()
	defer setStringP(&subnet, "a-subnet")()
	defer setStringP(&region, "a-region")()

	got := buildDaisyVars("translate/workflow/path")

	assert.Equal(t, got["image_name"], "image-a")
	assert.Equal(t, got["translate_workflow"], "translate/workflow/path")
	assert.Equal(t, got["install_gce_packages"], "false")
	assert.Equal(t, got["source_image"], "global/images/source-image")
	assert.Equal(t, got["family"], "a-family")
	assert.Equal(t, got["description"], "a-description")
	assert.Equal(t, got["import_network"], "global/networks/a-network")
	assert.Equal(t, got["import_subnet"], "regions/a-region/subnetworks/a-subnet")
	assert.Equal(t, len(got), 8)
}

func TestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(t *testing.T) {
	defer setStringP(&zone, "")()
	defer setStringP(&scratchBucketGcsPath, "gs://scratchbucket/scratchpath")()
	defer setStringP(&sourceFile, "gs://sourcebucket/sourcefile")()
	defer setStringP(&project, "a_project")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storageRegion := "europe-north1"

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone(storageRegion, *project).Return("europe-north1-b", nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: storageRegion}, nil)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-b", *zone)
	assert.Equal(t, "europe-north1", *region)
	assert.Equal(t, "gs://scratchbucket/scratchpath", *scratchBucketGcsPath)
}

func TestPopulateMissingParametersCreatesScratchBucketIfNotProvided(t *testing.T) {
	projectID := "a_project"
	defer setStringP(&zone, "")()
	defer setStringP(&scratchBucketGcsPath, "")()
	defer setStringP(&sourceFile, "gs://sourcebucket/sourcefile")()
	defer setStringP(&project, projectID)()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)

	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().
		CreateScratchBucket(*sourceFile, *project).
		Return("new_scratch_bucket", "europe-north1", nil).
		Times(1)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone("europe-north1", projectID).Return("europe-north1-c", nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", *zone)
	assert.Equal(t, "europe-north1", *region)
	assert.Equal(t, "gs://new_scratch_bucket/", *scratchBucketGcsPath)
}

func TestPopulateMissingParametersReturnsErrorWhenZoneCantBeRetrieved(t *testing.T) {
	projectID := "a_project"
	defer setStringP(&zone, "")()
	defer setStringP(&scratchBucketGcsPath, "gs://scratchbucket/scratchpath")()
	defer setStringP(&project, projectID)()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone("us-west2", projectID).Return("", fmt.Errorf("err")).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndNotRunningOnGCE(t *testing.T) {
	defer setStringP(&project, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndGCEProjectIdEmpty(t *testing.T) {
	defer setStringP(&project, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", nil)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateProjectIfMissingProjectPopulatedFromGCE(t *testing.T) {
	defer setStringP(&project, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("gce_project", nil)

	err := populateProjectIfMissing(mockMetadataGce)

	assert.Nil(t, err)
	assert.Equal(t, "gce_project", *project)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndMetadataReturnsError(t *testing.T) {
	defer setStringP(&project, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("pr", fmt.Errorf("Err"))
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketCreationError(t *testing.T) {
	defer setStringP(&scratchBucketGcsPath, "")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().CreateScratchBucket(*sourceFile, *project).Return("", "", fmt.Errorf("err"))
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketInvalidFormat(t *testing.T) {
	defer setStringP(&scratchBucketGcsPath, "NOT_GCS_PATH")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenPopulateRegionFails(t *testing.T) {
	defer setStringP(&zone, "NOT_ZONE")()
	defer setStringP(&scratchBucketGcsPath, "gs://scratchbucket/scratchpath")()
	defer setStringP(&sourceFile, "gs://sourcebucket/sourcefile")()
	defer setStringP(&project, "a_project")()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	storageRegion := "NOT_REGION"

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: storageRegion}, nil)

	err := populateMissingParameters(mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func createWorkflowWithCreateInstanceNetworkAccessConfig() *daisy.Workflow {
	w := daisy.New()
	w.Steps = map[string]*daisy.Step{
		"ci": {
			CreateInstances: &daisy.CreateInstances{
				{
					Instance: compute.Instance{
						Disks: []*compute.AttachedDisk{{Source: "key1"}},
						NetworkInterfaces: []*compute.NetworkInterface{
							{
								Network: "n",
								AccessConfigs: []*compute.AccessConfig{
									{Type: "ONE_TO_ONE_NAT"},
								},
							},
						},
					},
				},
			},
		},
	}
	return w
}

func backupOsArgs() func() {
	oldArgs := os.Args
	return func() { os.Args = oldArgs }
}

func buildOsArgs(cliArgs map[string]interface{}) {
	os.Args = make([]string, len(cliArgs)+1)
	i := 0
	os.Args[i] = "cmd"
	i++
	for key, value := range cliArgs {
		if value != nil {
			os.Args[i] = formatCliArg(key, value)
			i++
		}
	}
}

func formatCliArg(argKey, argValue interface{}) string {
	if argValue == true {
		return fmt.Sprintf("-%v", argKey)
	}
	if argValue != false {
		return fmt.Sprintf("-%v=%v", argKey, argValue)
	}
	return ""
}

func getAllCliArgs() map[string]interface{} {
	return map[string]interface{}{
		imageNameFlagKey:            "img",
		clientIDFlagKey:             "aClient",
		"data_disk":                 true,
		"os":                        "ubuntu-1404",
		"source_file":               "gs://source_bucket/source_file",
		"source_image":              "anImage",
		"no_guest_environment":      true,
		"family":                    "aFamily",
		"description":               "aDescription",
		"network":                   "aNetwork",
		"subnet":                    "aSubnet",
		"timeout":                   "2h",
		"zone":                      "us-central1-c",
		"project":                   "aProject",
		"scratch_bucket_gcs_path":   "gs://bucket/folder",
		"oauth":                     "oAuthFilePath",
		"compute_endpoint_override": "us-east1-c",
		"disable_gcs_logging":       true,
		"disable_cloud_logging":     true,
		"disable_stdout_logging":    true,
		"kms_key":                   "aKmsKey",
		"kms_keyring":               "aKmsKeyRing",
		"kms_location":              "aKmsLocation",
		"kms_project":               "aKmsProject",
		"labels":                    "userkey1=uservalue1,userkey2=uservalue2",
	}
}

func setStringP(p **string, value string) func() {
	oldValue := *p
	*p = &value
	return func() {
		*p = oldValue
	}
}

func setBoolP(p **bool, value bool) func() {
	oldValue := *p
	*p = &value
	return func() { *p = oldValue }
}

func clearStringFlag(cliArgs map[string]interface{}, flagKey string, flag **string) func() {
	delete(cliArgs, flagKey)
	return setStringP(flag, "")
}

func clearBoolFlag(cliArgs map[string]interface{}, flagKey string, flag **bool) func() {
	delete(cliArgs, flagKey)
	return setBoolP(flag, false)
}
