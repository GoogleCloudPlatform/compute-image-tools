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
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

const daisyWorkflows = "../../../../daisy_workflows"

func TestCreateInflater_File(t *testing.T) {
	inflater, err := newInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, &storage.Client{}, mockInspector{
		t:                 t,
		expectedReference: "gs://bucket/vmdk",
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{
			FileFormat:    "vpc",
		},
	}, nil)
	assert.NoError(t, err)
	facade, ok := inflater.(*inflaterFacade)
	assert.True(t, ok)

	mainInflater, ok := facade.mainInflater.(*daisyInflater)
	assert.True(t, ok)
	assert.Equal(t, "zones/us-west1-c/disks/disk-1234", mainInflater.inflatedDiskURI)
	assert.Equal(t, "gs://bucket/vmdk", mainInflater.wf.Vars["source_disk_file"].Value)
	assert.Equal(t, "projects/subnet/subnet", mainInflater.wf.Vars["import_subnet"].Value)
	assert.Equal(t, "projects/network/network", mainInflater.wf.Vars["import_network"].Value)

	network := getWorkerNetwork(t, mainInflater.wf)
	assert.Nil(t, network.AccessConfigs, "AccessConfigs must be nil to allow ExternalIP to be allocated.")

	realInflater, _ := facade.shadowInflater.(*apiInflater)
	assert.NotContains(t, realInflater.guestOsFeatures,
		&computeBeta.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestCreateInflater_Image(t *testing.T) {
	inflater, err := newInflater(ImageImportRequest{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
		WorkflowDir: daisyWorkflows,
	}, nil, &storage.Client{}, mockInspector{
		t:                 t,
		expectedReference: "",
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
	}, nil)
	assert.NoError(t, err)
	realInflater, ok := inflater.(*daisyInflater)
	assert.True(t, ok)
	assert.Equal(t, "zones/us-west1-b/disks/disk-1234", realInflater.inflatedDiskURI)
	assert.Equal(t, "projects/test/uri/image", realInflater.wf.Vars["source_image"].Value)
	inflatedDisk := getDisk(realInflater.wf, 0)
	assert.Contains(t, inflatedDisk.Licenses,
		"projects/compute-image-tools/global/licenses/virtual-disk-import")
}

func TestCreateAPIInflater_IncludesUEFIGuestOSFeature(t *testing.T) {
	request := ImageImportRequest{
		UefiCompatible: true,
	}
	realInflater, _ := createAPIInflater(request, nil, &storage.Client{}, logging.NewToolLogger(t.Name()), imagefile.Metadata{}).(*apiInflater)
	assert.Contains(t, realInflater.guestOsFeatures,
		&computeBeta.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestAPIInflater_Inflate_CreateDiskFailed_CancelWithoutDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDiskBeta(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to create disk"))
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled: cancel")

	inflater := createAPIInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, imagefile.Metadata{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.Error(t, err)
}

func TestAPIInflater_Inflate_CreateDiskSuccess_CancelWithDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDiskBeta(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().DeleteDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled: cancel")

	inflater := createAPIInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, imagefile.Metadata{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.NoError(t, err)
}

func TestAPIInflater_Inflate_Cancel_CleanupFailedToVerify(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("unknown"))

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled, cleanup failed to verify: cancel")

	inflater := createAPIInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, imagefile.Metadata{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
}

func TestAPIInflater_Inflate_Cancel_CleanupFailed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	mockLogger := mocks.NewMockLogger(mockCtrl)
	mockLogger.EXPECT().Debug("apiInflater.inflate is canceled, cleanup is failed: cancel")

	inflater := createAPIInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, mockComputeClient, &storage.Client{}, mockLogger, imagefile.Metadata{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
}

func TestAPIInflater_getCalculateChecksumWorkflow(t *testing.T) {
	inflater := createAPIInflater(ImageImportRequest{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, &storage.Client{}, logging.NewToolLogger(t.Name()), imagefile.Metadata{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	w := apiInflater.getCalculateChecksumWorkflow("")
	_, ok = w.Vars["compute_service_account"]
	assert.False(t, ok)

	apiInflater.request.ComputeServiceAccount = "email"
	w = apiInflater.getCalculateChecksumWorkflow("")
	assert.Equal(t, "email", w.Vars["compute_service_account"].Value)
}
