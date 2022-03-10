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
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/path"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

var (
	destinationURI, sourceImage, sourceDiskSnapshot, format, network, subnet, labels string
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

func TestFlagsDestinationUriNotProvided(t *testing.T) {
	resetArgs()
	destinationURI = ""
	assertErrorOnValidate("Expected error for missing destination_uri flag", t)
}

func assertErrorOnValidate(errorMsg string, t *testing.T) {
	if _, err := validateAndParseFlags(destinationURI, sourceImage, sourceDiskSnapshot, labels); err == nil {
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
		15,
		ws+format+ws,
		ws+network+ws,
		ws+subnet+ws,
		ws+"aRegion"+ws,
		"")

	assert.Equal(t, "global/images/anImage", got["source_image"])
	assert.Equal(t, "gs://bucket/exported_image", got["destination"])
	assert.Equal(t, "global/networks/aNetwork", got["export_network"])
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnet", got["export_subnet"])
	assert.Equal(t, "16", got["export_instance_disk_size"])
	assert.Equal(t, 5, len(got))
}

func TestBuildDaisyVarsWithFormatConversion(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+sourceImage+ws,
		ws+sourceDiskSnapshot+ws,
		5000,
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
	assert.Equal(t, "5250", got["export_instance_disk_size"])
	assert.Equal(t, 6, len(got))
}

func TestBuildDaisyVarsWithSimpleImageName(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		ws+destinationURI+ws,
		ws+"anImage"+ws,
		ws+""+ws,
		0,
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
		ws+"global/snapshots/aSnapshot"+ws,
		0,
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
		"", "", "", 0, "", "", "", "",
		ws+"account1"+ws)

	assert.Equal(t, "account1", got["compute_service_account"])
}

func TestBuildDaisyVarsWithoutComputeServiceAccount(t *testing.T) {
	resetArgs()
	ws := "\t \r\n\f\u0085\u00a0\u2000\u3000"
	got := buildDaisyVars(
		"", "", "", 0, "", "", "", "",
		ws)

	_, hasVar := got["compute_service_account"]
	assert.False(t, hasVar)
}

func TestValidateImageExists_ReturnsNoError_WhenImageByNameFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetImage("project", "image").Return(&v1.Image{DiskSizeGb: 21}, nil)
	diskSize, err := validateImageExists(mockComputeClient, "project", "image")
	assert.NoError(t, err)
	assert.Equal(t, int64(21), diskSize)
}

func TestValidateImageExists_ReturnsNoError_WhenImageByUriFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetImage("another-project", "image-name").Return(&v1.Image{DiskSizeGb: 22}, nil)
	diskSize, err := validateImageExists(mockComputeClient, "project", "projects/another-project/global/images/image-name")
	assert.NoError(t, err)
	assert.Equal(t, int64(22), diskSize)
}

func TestValidateImageExists_ReturnsError_WhenImageByNameNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetImage("project", "image").Return(nil, errors.New("image not found"))
	diskSize, err := validateImageExists(mockComputeClient, "project", "image")
	assert.EqualError(t, err,
		"Image \"image\" not found")
	assert.Equal(t, int64(0), diskSize)
}

func TestValidateImageExists_ReturnsError_WhenImageByUriNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetImage("another-project", "image-name").Return(nil, errors.New("image not found"))
	diskSize, err := validateImageExists(mockComputeClient, "project", "projects/another-project/global/images/image-name")
	assert.EqualError(t, err,
		"Image \"projects/another-project/global/images/image-name\" not found")
	assert.Equal(t, int64(0), diskSize)
}

func TestValidateSnapshotExists_ReturnsNoError_WhenSnapshotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetSnapshot("project", "snapshot").Return(&v1.Snapshot{DiskSizeGb: 21}, nil)
	diskSize, err := validateSnapshotExists(mockComputeClient, "project", "snapshot")
	assert.NoError(t, err)
	assert.Equal(t, int64(21), diskSize)
}

func TestValidateSnapshotExists_SkipsValidation_WhenSourceSnapshotIsValidURI(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	// No expectations on the mockComputeClient means the test fails if the mock detects calls.
	diskSize, err := validateSnapshotExists(mockComputeClient, "project", "projects/project/global/snapshot/snapshot-name")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), diskSize)
}

func TestValidateSnapshotExists_ReturnsError_WhenSnapshotNotFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetSnapshot("project", "snapshot").Return(nil, errors.New("snapshot not found"))
	diskSize, err := validateSnapshotExists(mockComputeClient, "project", "snapshot")
	assert.EqualError(t, err,
		"Snapshot \"snapshot\" not found")
	assert.Equal(t, int64(0), diskSize)
}

func resetArgs() {
	destinationURI = "gs://bucket/exported_image"
	sourceImage = "global/images/anImage"
	sourceDiskSnapshot = ""
	format = ""
	network = "aNetwork"
	subnet = "aSubnet"
	labels = "userkey1=uservalue1,userkey2=uservalue2"
}
