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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/imagefile"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/googleapi"
)

const daisyWorkflows = "../../../../daisy_workflows"

func TestCreateInflater_File(t *testing.T) {
	inflater, err := newInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	}, nil, storage.Client{}, mockInspector{
		t:                 t,
		expectedReference: "gs://bucket/vmdk",
		errorToReturn:     nil,
		metaToReturn:      imagefile.Metadata{},
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
	inflater, err := newInflater(ImportArguments{
		Source:      imageSource{uri: "projects/test/uri/image"},
		Zone:        "us-west1-b",
		ExecutionID: "1234",
		WorkflowDir: daisyWorkflows,
	}, nil, storage.Client{}, nil, nil)
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
	args := ImportArguments{
		UefiCompatible: true,
	}
	realInflater, _ := createAPIInflater(args, nil, storage.Client{}).(*apiInflater)
	assert.Contains(t, realInflater.guestOsFeatures,
		&computeBeta.GuestOsFeature{Type: "UEFI_COMPATIBLE"})
}

func TestAPIInflater_Inflate_CreateDiskFailed_CancelWithoutDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDiskBeta(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed to create disk"))
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	inflater := createAPIInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	},
		mockComputeClient,
		storage.Client{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.Equal(t, "apiInflater.inflate is canceled: cancel", apiInflater.serialLogs[0])
	assert.Error(t, err)
}

func TestAPIInflater_Inflate_CreateDiskSuccess_CancelWithDeleteDisk(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().CreateDiskBeta(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().DeleteDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, &googleapi.Error{Code: 404})

	inflater := createAPIInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	},
		mockComputeClient,
		storage.Client{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	// Send a cancel signal in prior to guarantee cancellation logic can be executed.
	cancelResult := apiInflater.Cancel("cancel")
	assert.True(t, cancelResult)

	_, _, err := apiInflater.Inflate()
	assert.Equal(t, "apiInflater.inflate is canceled: cancel", apiInflater.serialLogs[0])
	assert.NoError(t, err)
}

func TestAPIInflater_Inflate_Cancel_CleanupFailedToVerify(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("unknown"))

	inflater := createAPIInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	},
		mockComputeClient,
		storage.Client{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
	assert.Equal(t, "apiInflater.inflate is canceled, cleanup failed to verify: cancel", apiInflater.serialLogs[0])
}

func TestAPIInflater_Inflate_Cancel_CleanupFailed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)

	inflater := createAPIInflater(ImportArguments{
		Source:       fileSource{gcsPath: "gs://bucket/vmdk"},
		Subnet:       "projects/subnet/subnet",
		Network:      "projects/network/network",
		Zone:         "us-west1-c",
		ExecutionID:  "1234",
		NoExternalIP: false,
		WorkflowDir:  daisyWorkflows,
	},
		mockComputeClient,
		storage.Client{})

	apiInflater, ok := inflater.(*apiInflater)
	assert.True(t, ok)

	cancelResult := apiInflater.Cancel("cancel")
	assert.False(t, cancelResult)
	assert.Equal(t, "apiInflater.inflate is canceled, cleanup is failed: cancel", apiInflater.serialLogs[0])
}
