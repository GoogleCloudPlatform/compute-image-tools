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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func Test_MetadataProcessor_RecreateDiskWhenRequestedAttributesNotFound(t *testing.T) {
	project, zone := "test-project", "test-zone"
	originalDiskName, uefiDiskName := "disk-name", "disk-name-uefi"
	incomingDiskURI, uefiDiskURI := "zones/test-zone/disks/disk-name", "zones/test-zone/disks/disk-name-uefi"
	args := ImportArguments{
		Project: project,
		Zone:    zone,
	}
	tests := []struct {
		name             string
		fetchedDisk      *compute.Disk
		expectedNewDisk  *compute.Disk
		argPD            persistentDisk
		expectedReturnPD persistentDisk
	}{
		{
			name: "UEFI requested; returned disk not tagged",
			fetchedDisk: &compute.Disk{
				Name:       originalDiskName,
				SourceDisk: incomingDiskURI,
			},
			expectedNewDisk: &compute.Disk{
				Name:            uefiDiskName,
				SourceDisk:      incomingDiskURI,
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
			argPD:            persistentDisk{isUEFICompatible: true, uri: incomingDiskURI},
			expectedReturnPD: persistentDisk{isUEFICompatible: true, uri: uefiDiskURI},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl, mockComputeClient := createMockClient(t)
			defer mockCtrl.Finish()
			mockComputeClient.EXPECT().GetDisk(project, zone, originalDiskName).Return(tt.fetchedDisk, nil)
			mockComputeClient.EXPECT().CreateDisk(project, zone, tt.expectedNewDisk).Return(nil)
			mockComputeClient.EXPECT().DeleteDisk(project, zone, originalDiskName).Return(nil)

			returnedPD, err := newMetadataProcessor(mockComputeClient, args).process(tt.argPD, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedReturnPD, returnedPD)
		})
	}
}

func Test_MetadataProcessor_ReturnEarlyWhenNoChangesRequested(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()

	originalPD := persistentDisk{isUEFICompatible: false}
	returnedPD, err := newMetadataProcessor(mockComputeClient, ImportArguments{}).process(originalPD, nil)

	assert.NoError(t, err)
	assert.Equal(t, originalPD, returnedPD)
}

func Test_MetadataProcessor_DontModifyDisk_IfChangesAlreadyPresent(t *testing.T) {
	project, zone, originalDiskName := "test-project", "test-zone", "disk-name"
	args := ImportArguments{
		Project: project,
		Zone:    zone,
	}
	tests := []struct {
		name        string
		fetchedDisk *compute.Disk
		argPD       persistentDisk
	}{
		{
			name: "UEFI requested; returned disk already tagged",
			fetchedDisk: &compute.Disk{
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
			argPD: persistentDisk{isUEFICompatible: true, uri: "zones/test-zone/disks/disk-name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl, mockComputeClient := createMockClient(t)
			defer mockCtrl.Finish()
			mockComputeClient.EXPECT().GetDisk(project, zone, originalDiskName).Return(tt.fetchedDisk, nil)

			returnedPD, err := newMetadataProcessor(mockComputeClient, args).process(tt.argPD, nil)

			assert.NoError(t, err)
			assert.Equal(t, tt.argPD, returnedPD)
		})
	}
}

func Test_MetadataProcessor_DontModifyOriginalDisk_IfGetFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()

	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, errors.New("can't find disk"))
	argPD := persistentDisk{isUEFICompatible: true}
	returnedPD, err := newMetadataProcessor(mockComputeClient, ImportArguments{}).process(argPD, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get disk")
	assert.Equal(t, argPD, returnedPD)
}

func Test_MetadataProcessor_DontDeleteOriginalDisk_IfCreateFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, nil)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("disk create failed"))

	originalPD := persistentDisk{isUEFICompatible: true}
	returnedPD, err := newMetadataProcessor(mockComputeClient, ImportArguments{}).process(originalPD, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create UEFI disk")
	assert.Equal(t, originalPD, returnedPD)
}

func Test_MetadataProcessor_SilentlyPassesIfDeleteFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, nil)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().DeleteDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("can't find disk"))

	argPD := persistentDisk{isUEFICompatible: true, uri: "zones/test-zone/disks/disk-name"}
	expectedReturnPD := persistentDisk{isUEFICompatible: true, uri: "zones/test-zone/disks/disk-name-uefi"}
	returnedPD, err := newMetadataProcessor(mockComputeClient, ImportArguments{Zone: "test-zone"}).process(argPD, nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedReturnPD, returnedPD)
}

func createMockClient(t *testing.T) (*gomock.Controller, *mocks.MockClient) {
	mockCtrl := gomock.NewController(t)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	return mockCtrl, mockComputeClient
}
