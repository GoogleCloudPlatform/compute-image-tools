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

package exporter

import (
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/stretchr/testify/assert"
)

var (
	clientID, destinationURI, sourceImage, sourceDiskSnapshot, format, network, subnet, labels string
)

func TestGetWorkflowPathWithoutFormatConversion(t *testing.T) {
	resetArgs()
	workflow := getWorkflowPath(format, "")
	expectedWorkflow := path.ToWorkingDir(WorkflowDir+ExportWorkflow, "")
	if workflow != expectedWorkflow {
		t.Errorf("%v != %v", workflow, expectedWorkflow)
	}
}

func TestGetWorkflowPathWithFormatConversion(t *testing.T) {
	resetArgs()
	workflow := getWorkflowPath("vmdk", "")
	expectedWorkflow := path.ToWorkingDir(WorkflowDir+ExportAndConvertWorkflow, "")
	if workflow != expectedWorkflow {
		t.Errorf("%v != %v", workflow, expectedWorkflow)
	}
}

func TestFlagsBothSourceImageAndSourceSnapshotNotProvided(t *testing.T) {
	resetArgs()
	sourceImage = ""
	sourceDiskSnapshot = ""
	assertErrorOnValidate("Expected error for missing one of source_image and source_disk_snapshot flag", t)
}

func TestFlagsBothSourceImageAndSourceSnapshotProvided(t *testing.T) {
	resetArgs()
	sourceImage = "anImage"
	sourceDiskSnapshot = "aSnapshot"
	assertErrorOnValidate("Expected error for both source_image and source_disk_snapshot flags provided", t)
}

func TestFlagsClientIdNotProvided(t *testing.T) {
	resetArgs()
	clientID = ""
	assertErrorOnValidate("Expected error for missing client_id flag", t)
}

func TestFlagsDestinationUriNotProvided(t *testing.T) {
	resetArgs()
	destinationURI = ""
	assertErrorOnValidate("Expected error for missing destination_uri flag", t)
}

func assertErrorOnValidate(errorMsg string, t *testing.T) {
	if _, err := validateAndParseFlags(clientID, destinationURI, sourceImage, sourceDiskSnapshot, labels); err == nil {
		t.Error(errorMsg)
	}
}

func TestBuildDaisyVarsWithoutFormatConversion(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+sourceImage+ws,
		ws+sourceDiskSnapshot+ws,
		ws+format+ws,
		ws+network+ws,
		ws+subnet+ws,
		ws+"aRegion"+ws,
		"")

	assert.Equal(t, "global/images/anImage", got["source_image"])
	assert.Equal(t, "gs://bucket/exported_image", got["destination"])
	assert.Equal(t, "global/networks/aNetwork", got["export_network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", got["export_subnet"])
	assert.Equal(t, 4, len(got))
}

func TestBuildDaisyVarsWithFormatConversion(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+sourceImage+ws,
		ws+sourceDiskSnapshot+ws,
		ws+"vmdk"+ws,
		ws+network+ws,
		ws+subnet+ws,
		ws+"aRegion"+ws,
		"")

	assert.Equal(t, "global/images/anImage", got["source_image"])
	assert.Equal(t, "gs://bucket/exported_image", got["destination"])
	assert.Equal(t, "vmdk", got["format"])
	assert.Equal(t, "global/networks/aNetwork", got["export_network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", got["export_subnet"])
	assert.Equal(t, 5, len(got))
}

func TestBuildDaisyVarsWithSimpleImageName(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+"anImage"+ws,
		ws+""+ws,
		ws+format+ws,
		ws+network+ws,
		ws+subnet+ws,
		ws+"aRegion"+ws,
		"")

	assert.Equal(t, "global/images/anImage", got["source_image"])
}

func TestBuildDaisyVarsWithSimpleSnapshotName(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+""+ws,
		ws+"aSnapshot"+ws,
		ws+format+ws,
		ws+network+ws,
		ws+subnet+ws,
		ws+"aRegion"+ws,
		"")

	assert.Equal(t, "global/snapshots/aSnapshot", got["source_disk_snapshot"])
}

func TestBuildDaisyVarsWithComputeServiceAccount(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		"", "", "", "", "", "", "",
		ws+"account1"+ws)

	assert.Equal(t, "account1", got["compute_service_account"])
}

func TestBuildDaisyVarsWithoutComputeServiceAccount(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		"", "", "", "", "", "", "",
		ws)

	_, hasVar := got["compute_service_account"]
	assert.False(t, hasVar)
}

func resetArgs() {
	clientID = "aClient"
	destinationURI = "gs://bucket/exported_image"
	sourceImage = "global/images/anImage"
	sourceDiskSnapshot = ""
	format = ""
	network = "aNetwork"
	subnet = "aSubnet"
	labels = "userkey1=uservalue1,userkey2=uservalue2"
}
