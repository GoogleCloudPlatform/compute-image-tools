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

package importer

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

const daisyWorkflows = "../../../../daisy_workflows"

func TestCreateAPIInflater_IncludesUEFIGuestOSFeature(t *testing.T) {
	request := ImageImportRequest{
		UefiCompatible: true,
	}
	apiInflater := createAPIInflater(&apiInflaterProperties{request, nil, &storage.Client{}, logging.NewToolLogger(t.Name()), true, true})
	assert.Contains(t, apiInflater.guestOsFeatures,
		&compute.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestAPIInflater_ShadowInflate_CreateDiskFailed_CancelWithoutDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to create disk"))
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled: cancel")

	apiInflater := createAPIInflater(&apiInflaterProperties{ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, true, true})

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.Error(t, err)
}

func TestAPIInflater_ShadowInflate_CreateDiskSuccess_CancelWithDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().DeleteDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled: cancel")

	apiInflater := createAPIInflater(&apiInflaterProperties{ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, true, true})

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.NoError(t, err)
}

func TestAPIInflater_ShadowInflate_Cancel_CleanupFailedToVerify(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("unknown"))

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled, cleanup failed to verify: cancel")

	apiInflater := createAPIInflater(&apiInflaterProperties{ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, true, true})

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
}

func TestAPIInflater_ShadowInflate_Cancel_CleanupFailed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled, cleanup is failed: cancel")

	apiInflater := createAPIInflater(&apiInflaterProperties{ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, true, true})

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
}

func TestAPIInflater_getCalculateChecksumWorkflow(t *testing.T) {
	apiInflater := createAPIInflater(&apiInflaterProperties{ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, &storage.Client{}, logging.NewToolLogger(t.Name()), true, true})

	w := apiInflater.getCalculateChecksumWorkflow("", "shadow")
	assert.Equal(t, "default", w.Vars["compute_service_account"].Value)

	apiInflater.request.ComputeServiceAccount = "email"
	w = apiInflater.getCalculateChecksumWorkflow("", "shadow")
	assert.Equal(t, "email", w.Vars["compute_service_account"].Value)
}
