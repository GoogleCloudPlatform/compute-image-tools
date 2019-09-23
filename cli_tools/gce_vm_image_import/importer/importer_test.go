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

package importer

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	currentExecutablePath, clientID, imageName, osID, customTranWorkflow, sourceFile, sourceImage,
	family, description, network, subnet, labels string
	dataDisk, noGuestEnvironment bool
)

func TestGetWorkflowPathsFromImage(t *testing.T) {
	resetArgs()
	sourceImage = "image-1"
	osID = "ubuntu-1404"
	workflow, translate := getWorkflowPaths(dataDisk, osID, sourceImage, customTranWorkflow, currentExecutablePath)
	if workflow != path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath) || translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("%v != %v and/or translate not empty", workflow, WorkflowDir+ImportFromImageWorkflow)
	}
}

func TestGetWorkflowPathsDataDisk(t *testing.T) {
	resetArgs()
	dataDisk = true
	osID = ""
	sourceImage = ""
	workflow, translate := getWorkflowPaths(dataDisk, osID, sourceImage, customTranWorkflow, currentExecutablePath)
	if workflow != path.ToWorkingDir(WorkflowDir+ImportWorkflow, currentExecutablePath) || translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, WorkflowDir+ImportWorkflow)
	}
}

func TestGetWorkflowPathsWithCustomTranslateWorkflow(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	sourceImage = "image-1"
	customTranWorkflow = "custom.wf"
	osID = ""

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err != nil {

		t.Errorf("Unexpected flags error: %v", err)
	}
	workflow, translate := getWorkflowPaths(dataDisk, osID, sourceImage, customTranWorkflow, currentExecutablePath)
	if workflow != path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath) || translate != customTranWorkflow {
		t.Errorf("%v != %v and/or translate not empty", workflow, WorkflowDir+ImportFromImageWorkflow)
	}
}

func TestFlagsUnexpectedCustomTranslateWorkflowWithOs(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	customTranWorkflow = "custom.wf"

	_, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels)
	expected := fmt.Errorf("-os and -custom_translate_workflow can't be both specified")
	validateExpectedError(err, expected, t)
}

func TestFlagsUnexpectedCustomTranslateWorkflowWithDataDisk(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	dataDisk = true
	osID = ""
	customTranWorkflow = "custom.wf"

	_, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels)
	expected := fmt.Errorf("when -data_disk is specified, -os and -custom_translate_workflow should be empty")
	validateExpectedError(err, expected, t)
}

func TestGetWorkflowPathsFromFile(t *testing.T) {
	homeDir := "/home/gce/"

	resetArgs()
	imageName = "image-a"
	sourceImage = ""
	currentExecutablePath = homeDir + "executable"

	workflow, translate := getWorkflowPaths(dataDisk, osID, sourceImage, customTranWorkflow, currentExecutablePath)

	if workflow != homeDir+WorkflowDir+ImportAndTranslateWorkflow {
		t.Errorf("resulting workflow path `%v` does not match expected `%v`", workflow, homeDir+WorkflowDir+ImportAndTranslateWorkflow)
	}

	if translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("resulting translate workflow path `%v` does not match expected `%v`", translate, "ubuntu/translate_ubuntu_1404.wf.json")
	}
}

func TestFlagsImageNameNotProvided(t *testing.T) {
	resetArgs()
	imageName = ""
	_, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels)
	expected := fmt.Errorf("The flag -image_name must be provided")
	validateExpectedError(err, expected, t)
}

func assertErrorOnValidate(errorMsg string, t *testing.T) {
	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err == nil {
		t.Error(errorMsg)
	}
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	resetArgs()
	clientID = ""
	assertErrorOnValidate("Expected error for missing client_id flag", t)
}

func TestFlagsDataDiskOrOSFlagsNotProvided(t *testing.T) {
	resetArgs()
	osID = ""
	dataDisk = false
	assertErrorOnValidate("Expected error for missing os or data_disk flag", t)
}

func TestFlagsDataDiskAndOSFlagsBothProvided(t *testing.T) {
	resetArgs()
	dataDisk = true
	assertErrorOnValidate("Expected error for both os and data_disk set at the same time", t)
}

func TestFlagsSourceFileOrSourceImageNotProvided(t *testing.T) {
	resetArgs()
	sourceFile = ""
	sourceImage = ""
	dataDisk = false
	assertErrorOnValidate("Expected error for missing source_file or source_image flag", t)
}

func TestFlagsSourceFileAndSourceImageBothProvided(t *testing.T) {
	resetArgs()
	sourceFile = "gs://source_bucket/source_file"
	dataDisk = false
	assertErrorOnValidate("Expected error for both source_file and source_image flags set", t)
}

func TestFlagsSourceFile(t *testing.T) {
	resetArgs()
	sourceFile = "gs://source_bucket/source_file"
	sourceImage = ""
	dataDisk = false

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagSourceFileCompressed(t *testing.T) {
	fileString := test.CreateCompressedFile()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetObjectReader(gomock.Any(), gomock.Any()).Return(ioutil.NopCloser(strings.NewReader(fileString)), nil)

	err := validateSourceFile(mockStorageClient, "", "")
	assert.NotNil(t, err, "Expected error")
}

func TestFlagSourceFileUncompressed(t *testing.T) {
	fileString := "random content"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetObjectReader(gomock.Any(), gomock.Any()).Return(ioutil.NopCloser(strings.NewReader(fileString)), nil)

	err := validateSourceFile(mockStorageClient, "", "")
	assert.Nil(t, err, "Unexpected error")
}

func TestFlagsInvalidSourceFile(t *testing.T) {
	resetArgs()
	sourceFile = "invalidSourceFile"
	sourceImage = ""

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err == nil {
		t.Errorf("Expected error")
	}
}

func TestFlagsSourceImage(t *testing.T) {
	resetArgs()
	sourceFile = ""

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsDataDisk(t *testing.T) {
	resetArgs()
	sourceFile = "gs://source_bucket/source_file"
	sourceImage = ""
	osID = ""
	dataDisk = true

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidOS(t *testing.T) {
	resetArgs()
	sourceFile = "gs://source_bucket/source_file"
	sourceImage = ""
	osID = "invalidOs"

	if _, _, _, err := validateAndParseFlags(clientID, imageName, sourceFile, sourceImage, dataDisk,
		osID, customTranWorkflow, labels); err == nil {
		t.Errorf("Expected error")
	}
}

func TestBuildDaisyVarsFromDisk(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	noGuestEnvironment = true
	sourceFile = "source-file-path"
	sourceImage = ""
	family = "a-family"
	description = "a-description"
	network = "a-network"
	subnet = "a-subnet"
	region := "a-region"

	got := buildDaisyVars("translate/workflow/path", imageName, sourceFile,
		sourceImage, family, description, region, subnet, network, noGuestEnvironment)

	assert.Equal(t, got["image_name"], "image-a")
	assert.Equal(t, got["translate_workflow"], "translate/workflow/path")
	assert.Equal(t, got["install_gce_packages"], "false")
	assert.Equal(t, got["source_disk_file"], "source-file-path")
	assert.Equal(t, got["family"], "a-family")
	assert.Equal(t, got["description"], "a-description")
	assert.Equal(t, got["import_network"], "global/networks/a-network")
	assert.Equal(t, got["import_subnet"], "regions/a-region/subnetworks/a-subnet")
	assert.Equal(t, got["IsWindows"], "false")
	assert.Equal(t, len(got), 9)
}

func TestBuildDaisyVarsFromImage(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	noGuestEnvironment = true
	sourceFile = ""
	sourceImage = "source-image"
	family = "a-family"
	description = "a-description"
	network = "a-network"
	subnet = "a-subnet"
	region := "a-region"

	got := buildDaisyVars("translate/workflow/path", imageName, sourceFile,
		sourceImage, family, description, region, subnet, network, noGuestEnvironment)

	assert.Equal(t, got["image_name"], "image-a")
	assert.Equal(t, got["translate_workflow"], "translate/workflow/path")
	assert.Equal(t, got["install_gce_packages"], "false")
	assert.Equal(t, got["source_image"], "global/images/source-image")
	assert.Equal(t, got["family"], "a-family")
	assert.Equal(t, got["description"], "a-description")
	assert.Equal(t, got["import_network"], "global/networks/a-network")
	assert.Equal(t, got["import_subnet"], "regions/a-region/subnetworks/a-subnet")
	assert.Equal(t, got["IsWindows"], "false")
	assert.Equal(t, len(got), 9)
}

func TestBuildDaisyVarsWindow(t *testing.T) {
	resetArgs()
	imageName = "image-a"

	region := ""
	got := buildDaisyVars("translate/workflow/path/windows", imageName, sourceFile,
		sourceImage, family, description, region, subnet, network, noGuestEnvironment)

	assert.Equal(t, "true", got["IsWindows"])
}

func TestBuildDaisyVarsImageNameLowercase(t *testing.T) {
	resetArgs()
	imageName = "IMAGE-a"

	region := ""
	got := buildDaisyVars("translate/workflow/path", imageName, sourceFile,
		sourceImage, family, description, region, subnet, network, noGuestEnvironment)

	assert.Equal(t, got["image_name"], "image-a")
}

func validateExpectedError(err error, expected error, t *testing.T) {
	if err != expected {
		if err == nil {
			t.Errorf("nil != %v", expected)
		} else if err.Error() != expected.Error() {
			t.Errorf("%v != %v", err, expected)
		}
	}
}

func resetArgs() {
	sourceFile = ""
	sourceImage = "anImage"
	osID = "ubuntu-1404"
	dataDisk = false
	imageName = "img"
	clientID = "aClient"
	customTranWorkflow = ""
	currentExecutablePath = ""
	labels = "userkey1=uservalue1,userkey2=uservalue2"
}
