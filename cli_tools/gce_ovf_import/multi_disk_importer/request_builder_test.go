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

package multidiskimporter

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	imagemocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer/mocks"
	daisyovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/daisy_utils"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
)

func TestBuildRequests_InitsFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileURIs := []string{"gs://bucket/disk1Request.vmdk", "gs://bucket/disk2Request.vmdk"}
	params := makeDefaultParams()
	params.InstanceNames = "xyz"

	builder := &requestBuilder{
		workflowDir:   "/daisy/workflows",
		sourceFactory: initSourceFactory(ctrl, fileURIs),
	}

	requests, actualError := builder.buildRequests(params, fileURIs)
	assert.NoError(t, actualError)
	assert.Len(t, requests, len(fileURIs))

	disk1Request := requests[0]
	disk2Request := requests[1]

	assert.Equal(t, disk1Request.DiskName, daisyovfutils.GenerateDataDiskName(params.InstanceNames, 0))
	assert.Equal(t, disk2Request.DiskName, daisyovfutils.GenerateDataDiskName(params.InstanceNames, 1))

	assert.Equal(t, fileURIs[0], disk1Request.Source.Path())
	assert.Equal(t, fileURIs[1], disk2Request.Source.Path())

	assert.True(t, disk1Request.DataDisk)
	assert.True(t, disk2Request.DataDisk)

	assert.Equal(t, fmt.Sprintf("%s/%s", params.ScratchBucketGcsPath, disk1Request.DiskName), disk1Request.ScratchBucketGcsPath)
	assert.Equal(t, fmt.Sprintf("%s/%s", params.ScratchBucketGcsPath, disk2Request.DiskName), disk2Request.ScratchBucketGcsPath)

	assert.Equal(t, "disk-1", disk1Request.DaisyLogLinePrefix)
	assert.Equal(t, "disk-2", disk2Request.DaisyLogLinePrefix)

	assertAllEqual(t, params.CloudLogsDisabled, disk1Request.CloudLogsDisabled, disk2Request.CloudLogsDisabled)
	assertAllEqual(t, params.GcsLogsDisabled, disk1Request.GcsLogsDisabled, disk2Request.GcsLogsDisabled)
	assertAllEqual(t, params.StdoutLogsDisabled, disk1Request.StdoutLogsDisabled, disk2Request.StdoutLogsDisabled)
	assertAllEqual(t, params.UefiCompatible, disk1Request.UefiCompatible, disk2Request.UefiCompatible)
	assertAllEqual(t, params.NoExternalIP, disk1Request.NoExternalIP, disk2Request.NoExternalIP)
	assertAllEqual(t, params.Ce, disk1Request.ComputeEndpoint, disk2Request.ComputeEndpoint)
	assertAllEqual(t, params.Oauth, disk1Request.Oauth, disk2Request.Oauth)
	assertAllEqual(t, params.Network, disk1Request.Network, disk2Request.Network)
	assertAllEqual(t, params.Subnet, disk1Request.Subnet, disk2Request.Subnet)
	assertAllEqual(t, params.Zone, disk1Request.Zone, disk2Request.Zone)
	assertAllEqual(t, *params.Project, disk1Request.Project, disk2Request.Project)
	assertAllEqual(t, builder.workflowDir, disk1Request.WorkflowDir, disk2Request.WorkflowDir)
}

func TestBuildRequests_CreatesTimeout_FromDeadline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileURIs := []string{"gs://bucket/disk1Request.vmdk"}
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

func TestBuildRequests_ReturnError_WhenSourceInitializationFails(t *testing.T) {
	initError := errors.New("failed to init source")
	fileURIs := []string{"gs://bucket/disk1Request.vmdk", "gs://bucket/disk2Request.vmdk"}
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
