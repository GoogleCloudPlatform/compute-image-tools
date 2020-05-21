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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
)

var (
	currentExecutablePath, imageName, osID, customTranWorkflow,
	family, description, network, subnet string
	dataDisk, noGuestEnvironment, sysprepWindows bool
	source                                       Source
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

	workflow, translate := getWorkflowPaths(source, dataDisk, osID, customTranWorkflow, currentExecutablePath)
	assert.Equal(t, path.ToWorkingDir(WorkflowDir+ImportFromImageWorkflow, currentExecutablePath), workflow)
	assert.Equal(t, translate, customTranWorkflow)
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

func resetArgs() {
	source = imageSource{uri: "global/images/source-image"}
	osID = "ubuntu-1404"
	dataDisk = false
	sysprepWindows = false
	imageName = "img"
	customTranWorkflow = ""
	currentExecutablePath = ""
}
