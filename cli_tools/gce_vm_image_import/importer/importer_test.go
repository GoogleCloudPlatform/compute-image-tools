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
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/stretchr/testify/assert"
)

var (
	currentExecutablePath, clientID, imageName, osID, customTranWorkflow,
	family, description, network, subnet, labels string
	dataDisk, noGuestEnvironment, sysprepWindows bool
	source                                       resource
)

func TestGetWorkflowPathsFromImage(t *testing.T) {
	resetArgs()
	source = imageSource{uri: "uri"}
	osID = "ubuntu-1404"
	workflow, translate := getWorkflowPaths(source, dataDisk, osID, customTranWorkflow, currentExecutablePath)
	if workflow != path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath) || translate != "ubuntu/translate_ubuntu_1404.wf.json" {
		t.Errorf("%v != %v and/or translate not empty", workflow, WorkflowDir+ImportFromImageWorkflow)
	}
}

func TestGetWorkflowPathsDataDisk(t *testing.T) {
	resetArgs()
	dataDisk = true
	osID = ""
	source = fileSource{}
	workflow, translate := getWorkflowPaths(source, dataDisk, osID, customTranWorkflow, currentExecutablePath)
	if workflow != path.ToWorkingDir(WorkflowDir+ImportWorkflow, currentExecutablePath) || translate != "" {
		t.Errorf("%v != %v and/or translate not empty", workflow, WorkflowDir+ImportWorkflow)
	}
}

func TestGetWorkflowPathsWithCustomTranslateWorkflow(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	source = imageSource{}
	customTranWorkflow = "custom.wf"
	osID = ""

	if _, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels); err != nil {

		t.Errorf("Unexpected flags error: %v", err)
	}
	workflow, translate := getWorkflowPaths(source, dataDisk, osID, customTranWorkflow, currentExecutablePath)
	assert.Equal(t, path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath), workflow)
	assert.Equal(t, translate, customTranWorkflow)
}

func TestFlagsUnexpectedCustomTranslateWorkflowWithOs(t *testing.T) {
	resetArgs()
	imageName = "image-a"
	customTranWorkflow = "custom.wf"

	_, err := validateAndParseFlags(clientID, imageName, dataDisk,
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

	_, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels)
	expected := fmt.Errorf("when -data_disk is specified, -os and -custom_translate_workflow should be empty")
	validateExpectedError(err, expected, t)
}

func TestGetWorkflowPathsFromFile(t *testing.T) {
	homeDir := "/home/gce/"

	resetArgs()
	imageName = "image-a"
	source = fileSource{}
	currentExecutablePath = homeDir + "executable"

	workflow, translate := getWorkflowPaths(source, dataDisk, osID, customTranWorkflow, currentExecutablePath)

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
	_, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels)
	expected := fmt.Errorf("The flag -image_name must be provided")
	validateExpectedError(err, expected, t)
}

func assertErrorOnValidate(errorMsg string, t *testing.T) {
	if _, err := validateAndParseFlags(clientID, imageName, dataDisk,
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

func TestFlagsSourceFile(t *testing.T) {
	resetArgs()
	dataDisk = false

	if _, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsDataDisk(t *testing.T) {
	resetArgs()
	osID = ""
	dataDisk = true

	if _, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestFlagsInvalidOS(t *testing.T) {
	resetArgs()
	osID = "invalidOs"

	if _, err := validateAndParseFlags(clientID, imageName, dataDisk,
		osID, customTranWorkflow, labels); err == nil {
		t.Errorf("Expected error")
	}
}

func TestBuildDaisyVarsFromDisk(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	imageName = ws + "image-a" + ws
	noGuestEnvironment = true
	source = fileSource{gcsPath: "source-file-path"}
	family = ws + "a-family" + ws
	description = ws + "a-description" + ws
	network = ws + "a-network" + ws
	subnet = ws + "a-subnet" + ws
	region := ws + "a-region" + ws

	got := buildDaisyVars(source, "translate/workflow/path", imageName,
		family, description, region, subnet, network, noGuestEnvironment, sysprepWindows)

	assert.Equal(t, "image-a", got["image_name"])
	assert.Equal(t, "translate/workflow/path", got["translate_workflow"])
	assert.Equal(t, "false", got["install_gce_packages"])
	assert.Equal(t, "source-file-path", got["source_disk_file"])
	assert.Equal(t, "a-family", got["family"])
	assert.Equal(t, "a-description", got["description"])
	assert.Equal(t, "global/networks/a-network", got["import_network"])
	assert.Equal(t, "regions/a-region/subnetworks/a-subnet", got["import_subnet"])
	assert.Equal(t, "false", got["is_windows"])
	assert.Equal(t, "false", got["sysprep_windows"])
	assert.Equal(t, 10, len(got))
}

func TestBuildDaisyVarsFromImage(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	imageName = ws + "image-a" + ws
	noGuestEnvironment = true
	source = imageSource{uri: "global/images/source-image"}
	family = ws + "a-family" + ws
	description = ws + "a-description" + ws
	network = ws + "a-network" + ws
	subnet = ws + "a-subnet" + ws
	region := ws + "a-region" + ws

	got := buildDaisyVars(source, "translate/workflow/path", imageName,
		family, description, region, subnet, network, noGuestEnvironment, sysprepWindows)

	assert.Equal(t, "image-a", got["image_name"])
	assert.Equal(t, "translate/workflow/path", got["translate_workflow"])
	assert.Equal(t, "false", got["install_gce_packages"])
	assert.Equal(t, "global/images/source-image", got["source_image"])
	assert.Equal(t, "a-family", got["family"])
	assert.Equal(t, "a-description", got["description"])
	assert.Equal(t, "global/networks/a-network", got["import_network"])
	assert.Equal(t, "regions/a-region/subnetworks/a-subnet", got["import_subnet"])
	assert.Equal(t, "false", got["is_windows"])
	assert.Equal(t, "false", got["sysprep_windows"])
	assert.Equal(t, 10, len(got))
}

func TestBuildDaisyVarsWindowsSysprepEnabled(t *testing.T) {
	resetArgs()
	sysprepWindows = true
	got := buildDaisyVars(source, "translate/workflow/path/windows", "image-a",
		family, description, "", subnet, network, noGuestEnvironment, sysprepWindows)

	assert.Equal(t, "true", got["sysprep_windows"])
}

func TestBuildDaisyVarsWindowsSysprepDisabled(t *testing.T) {
	resetArgs()
	sysprepWindows = false
	got := buildDaisyVars(source, "translate/workflow/path/windows", "image-a",
		family, description, "", subnet, network, noGuestEnvironment, sysprepWindows)

	assert.Equal(t, "false", got["sysprep_windows"])
}

func TestBuildDaisyVarsIsWindows(t *testing.T) {
	resetArgs()
	imageName = "image-a"

	region := ""
	got := buildDaisyVars(source, "translate/workflow/path/windows", imageName,
		family, description, region, subnet, network, noGuestEnvironment, sysprepWindows)

	assert.Equal(t, "true", got["is_windows"])
}

func TestBuildDaisyVarsImageNameLowercase(t *testing.T) {
	resetArgs()
	imageName = "IMAGE-a"

	region := ""
	got := buildDaisyVars(source, "translate/workflow/path", imageName,
		family, description, region, subnet, network, noGuestEnvironment, sysprepWindows)

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
	source = imageSource{uri: "global/images/source-image"}
	osID = "ubuntu-1404"
	dataDisk = false
	sysprepWindows = false
	imageName = "img"
	clientID = "aClient"
	customTranWorkflow = ""
	currentExecutablePath = ""
	labels = "userkey1=uservalue1,userkey2=uservalue2"
}
