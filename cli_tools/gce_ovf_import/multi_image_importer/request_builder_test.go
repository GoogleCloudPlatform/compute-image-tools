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

package multiimageimporter

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	imagemocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer/mocks"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
)

func TestBuildRequests_InitsFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileURIs := []string{"gs://bucket/disk1.vmdk", "gs://bucket/disk2.vmdk"}
	params := makeDefaultParams()
	params.BuildID = "xyz"

	builder := &requestBuilder{
		workflowDir:   "/daisy/workflows",
		sourceFactory: initSourceFactory(ctrl, fileURIs),
	}
	requests, actualError := builder.buildRequests(params, fileURIs)
	assert.NoError(t, actualError)
	assert.Len(t, requests, len(fileURIs))

	boot := requests[0]
	data := requests[1]

	assert.Equal(t, "ovf-xyz-1", boot.ExecutionID)
	assert.Equal(t, "ovf-xyz-2", data.ExecutionID)

	assert.Equal(t, fileURIs[0], boot.Source.Path())
	assert.Equal(t, fileURIs[1], data.Source.Path())

	assert.Equal(t, params.OsID, boot.OS)
	assert.Empty(t, data.OS)

	assert.False(t, boot.DataDisk)
	assert.True(t, data.DataDisk)

	assert.Equal(t, "ovf-xyz-1", boot.ImageName)
	assert.Equal(t, "ovf-xyz-2", data.ImageName)

	assert.Equal(t, params.ScratchBucketGcsPath+"/ovf-xyz-1", boot.ScratchBucketGcsPath)
	assert.Equal(t, params.ScratchBucketGcsPath+"/ovf-xyz-2", data.ScratchBucketGcsPath)

	assert.Equal(t, "disk-1", boot.DaisyLogLinePrefix)
	assert.Equal(t, "disk-2", data.DaisyLogLinePrefix)

	assertAllEqual(t, params.CloudLogsDisabled, boot.CloudLogsDisabled, data.CloudLogsDisabled)
	assertAllEqual(t, params.GcsLogsDisabled, boot.GcsLogsDisabled, data.GcsLogsDisabled)
	assertAllEqual(t, params.StdoutLogsDisabled, boot.StdoutLogsDisabled, data.StdoutLogsDisabled)
	assertAllEqual(t, params.UefiCompatible, boot.UefiCompatible, data.UefiCompatible)
	assertAllEqual(t, params.NoExternalIP, boot.NoExternalIP, data.NoExternalIP)
	assertAllEqual(t, params.NoGuestEnvironment, boot.NoGuestEnvironment, data.NoGuestEnvironment)
	assertAllEqual(t, params.Ce, boot.ComputeEndpoint, data.ComputeEndpoint)
	assertAllEqual(t, params.Oauth, boot.Oauth, data.Oauth)
	assertAllEqual(t, params.Network, boot.Network, data.Network)
	assertAllEqual(t, params.Subnet, boot.Subnet, data.Subnet)
	assertAllEqual(t, params.Zone, boot.Zone, data.Zone)
	assertAllEqual(t, *params.Project, boot.Project, data.Project)
	assertAllEqual(t, builder.workflowDir, boot.WorkflowDir, data.WorkflowDir)

}

func TestBuildRequests_CreatesTimeout_FromDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileURIs := []string{"gs://bucket/disk1.vmdk"}
	expectedTimeout := time.Minute * 22
	params := makeDefaultParams()
	params.Deadline = time.Now().Add(expectedTimeout)
	requests, actualError := (&requestBuilder{
		sourceFactory: initSourceFactory(ctrl, fileURIs),
	}).buildRequests(params, fileURIs)
	assert.NoError(t, actualError)
	assert.Len(t, requests, len(fileURIs))

	actualTimeout := requests[0].Timeout
	timeoutDiff := time.Duration(math.Abs(float64(actualTimeout - expectedTimeout)))
	if timeoutDiff > time.Second*5 {
		t.Errorf("expectedTimeout=%s, actualTimeout=%s", expectedTimeout, actualTimeout)
	}
}

func TestBuildRequests_ReturnError_WhenTimeoutExceeded(t *testing.T) {
	params := makeDefaultParams()
	params.Deadline = time.Now()
	_, actualError := (&requestBuilder{}).buildRequests(params, []string{"gs://bucket/disk1.vmdk"})
	assert.EqualError(t, actualError, "Timeout exceeded")
}

func TestBuildRequests_ReturnError_WhenSourceInitializationFails(t *testing.T) {
	initError := errors.New("failed to init source")
	fileURIs := []string{"gs://bucket/disk1.vmdk", "gs://bucket/disk2.vmdk"}
	params := makeDefaultParams()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSourceFactory := imagemocks.NewMockSourceFactory(ctrl)
	mockSourceFactory.EXPECT().Init(gomock.Any(), "").Return(nil, initError).AnyTimes()

	_, actualError := (&requestBuilder{
		workflowDir:   "/path/to/daisy_workflows",
		sourceFactory: mockSourceFactory,
	}).buildRequests(params, fileURIs)
	assert.Equal(t, actualError, initError)
}
func initSourceFactory(ctrl *gomock.Controller, fileURIs []string) importer.SourceFactory {
	mockSourceFactory := imagemocks.NewMockSourceFactory(ctrl)
	for _, fileURI := range fileURIs {
		mockSourceFactory.EXPECT().Init(fileURI, "").Return(&fakeSource{fileURI}, nil)
	}
	return mockSourceFactory
}

func assertAllEqual(t *testing.T, values ...interface{}) {
	first := values[0]
	for _, value := range values {
		if value != first {
			t.Fatalf("Expected %s to equal %s", first, value)
		}
	}
}

type fakeSource struct {
	fileURI string
}

func (f *fakeSource) Path() string {
	return f.fileURI
}

func makeDefaultParams() *ovfdomain.OVFImportParams {
	project := "test-project"
	return &ovfdomain.OVFImportParams{
		Deadline:             time.Now().Add(time.Hour),
		Project:              &project,
		ScratchBucketGcsPath: "gs://scratch/bucket",
		CloudLogsDisabled:    true,
		Ce:                   "https://compute-endpoint",
		GcsLogsDisabled:      true,
		Network:              "global/network",
		NoExternalIP:         true,
		NoGuestEnvironment:   true,
		Oauth:                "oauth token",
		OsID:                 "ubuntu-1804",
		StdoutLogsDisabled:   true,
		Subnet:               "regional/subnet",
		UefiCompatible:       true,
		Zone:                 "us-central1-a",
	}
}
