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
	originalDiskName, modifiedDiskName := "disk-name", "disk-name-1"
	incomingDiskURI, modifiedDiskURI := "zones/test-zone/disks/disk-name", "zones/test-zone/disks/disk-name-1"
	tests := []struct {
		name             string
		fetchedDisk      *compute.Disk
		expectedNewDisk  *compute.Disk
		argPD            persistentDisk
		requiredFeatures []*compute.GuestOsFeature
		requiredLicenses []string
		expectedReturnPD persistentDisk
	}{
		{
			name: "Needs one guest OS feature",
			fetchedDisk: &compute.Disk{
				Name:       originalDiskName,
				SourceDisk: incomingDiskURI,
			},
			expectedNewDisk: &compute.Disk{
				Name:            modifiedDiskName,
				SourceDisk:      incomingDiskURI,
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
			requiredFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			argPD:            persistentDisk{uri: incomingDiskURI},
			expectedReturnPD: persistentDisk{uri: modifiedDiskURI},
		},
		{
			name: "Needs two guest OS feature",
			fetchedDisk: &compute.Disk{
				Name:       originalDiskName,
				SourceDisk: incomingDiskURI,
			},
			expectedNewDisk: &compute.Disk{
				Name:            modifiedDiskName,
				SourceDisk:      incomingDiskURI,
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "WINDOWS"}, {Type: "UEFI_COMPATIBLE"}},
			},
			requiredFeatures: []*compute.GuestOsFeature{{Type: "WINDOWS"}, {Type: "UEFI_COMPATIBLE"}},
			argPD:            persistentDisk{uri: incomingDiskURI},
			expectedReturnPD: persistentDisk{uri: modifiedDiskURI},
		},
		{
			name: "Needs license",
			fetchedDisk: &compute.Disk{
				Name:       originalDiskName,
				SourceDisk: incomingDiskURI,
				Licenses:   []string{"existing/license"},
			},
			expectedNewDisk: &compute.Disk{
				Name:       modifiedDiskName,
				SourceDisk: incomingDiskURI,
				Licenses:   []string{"existing/license", "additional/license/uri"},
			},
			requiredLicenses: []string{"additional/license/uri"},
			argPD:            persistentDisk{uri: incomingDiskURI},
			expectedReturnPD: persistentDisk{uri: modifiedDiskURI},
		},
		{
			name: "Needs license and UEFI tag",
			fetchedDisk: &compute.Disk{
				Name:       originalDiskName,
				SourceDisk: incomingDiskURI,
				Licenses:   []string{"existing/license"},
			},
			expectedNewDisk: &compute.Disk{
				Name:            modifiedDiskName,
				SourceDisk:      incomingDiskURI,
				Licenses:        []string{"existing/license", "additional/license/uri"},
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
			requiredFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			requiredLicenses: []string{"additional/license/uri"},
			argPD:            persistentDisk{uri: incomingDiskURI},
			expectedReturnPD: persistentDisk{uri: modifiedDiskURI},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl, mockComputeClient := createMockClient(t)
			defer mockCtrl.Finish()
			mockComputeClient.EXPECT().GetDisk(project, zone, originalDiskName).Return(tt.fetchedDisk, nil)
			mockComputeClient.EXPECT().CreateDisk(project, zone, tt.expectedNewDisk).Return(nil)
			mockComputeClient.EXPECT().DeleteDisk(project, zone, originalDiskName).Return(nil)

			processor := newMetadataProcessor(project, zone, mockComputeClient)
			processor.requiredLicenses = tt.requiredLicenses
			processor.requiredFeatures = tt.requiredFeatures
			returnedPD, err := processor.process(tt.argPD, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedReturnPD, returnedPD)
		})
	}
}

func Test_MetadataProcessor_ReturnEarlyWhenNoChangesRequested(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()

	originalPD := persistentDisk{}
	returnedPD, err := newMetadataProcessor("project", "zone", mockComputeClient).process(originalPD, nil)

	assert.NoError(t, err)
	assert.Equal(t, originalPD, returnedPD)
}

func Test_MetadataProcessor_DontModifyDisk_IfChangesAlreadyPresent(t *testing.T) {
	project, zone, originalDiskName := "test-project", "test-zone", "disk-name"
	tests := []struct {
		name             string
		requiredLicenses []string
		requiredFeatures []*compute.GuestOsFeature
		fetchedDisk      *compute.Disk
		argPD            persistentDisk
	}{
		{
			name: "UEFI requested; already tagged.",
			fetchedDisk: &compute.Disk{
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			},
			requiredFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			argPD:            persistentDisk{uri: "zones/test-zone/disks/disk-name"},
		},
		{
			name: "Requested license already present.",
			fetchedDisk: &compute.Disk{
				Licenses: []string{"other/license", "requested/license"},
			},
			requiredLicenses: []string{"requested/license"},
			argPD:            persistentDisk{uri: "zones/test-zone/disks/disk-name"},
		},
		{
			name: "UEFI and license requested; both present.",
			fetchedDisk: &compute.Disk{
				GuestOsFeatures: []*compute.GuestOsFeature{{Type: "WINDOWS"}, {Type: "UEFI_COMPATIBLE"}},
				Licenses:        []string{"other/license", "requested/license"},
			},
			requiredFeatures: []*compute.GuestOsFeature{{Type: "UEFI_COMPATIBLE"}},
			requiredLicenses: []string{"requested/license"},
			argPD:            persistentDisk{uri: "zones/test-zone/disks/disk-name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl, mockComputeClient := createMockClient(t)
			defer mockCtrl.Finish()
			mockComputeClient.EXPECT().GetDisk(project, zone, originalDiskName).Return(tt.fetchedDisk, nil)

			processor := newMetadataProcessor(project, zone, mockComputeClient)
			processor.requiredLicenses = tt.requiredLicenses
			processor.requiredFeatures = tt.requiredFeatures
			returnedPD, err := processor.process(tt.argPD, nil)

			assert.NoError(t, err)
			assert.Equal(t, tt.argPD, returnedPD)
		})
	}
}

func Test_MetadataProcessor_DontModifyOriginalDisk_IfGetFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()

	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, errors.New("can't find disk"))
	argPD := persistentDisk{sizeGb: 10}
	processor := newMetadataProcessor("project", "zone", mockComputeClient)
	processor.requiredLicenses = []string{"new/license"}
	returnedPD, err := processor.process(argPD, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to get disk")
	assert.Equal(t, argPD, returnedPD)
}

func Test_MetadataProcessor_DontDeleteOriginalDisk_IfCreateFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, nil)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("disk create failed"))

	argPD := persistentDisk{sizeGb: 10}
	processor := newMetadataProcessor("project", "zone", mockComputeClient)
	processor.requiredLicenses = []string{"new/license"}
	returnedPD, err := processor.process(argPD, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to create UEFI disk")
	assert.Equal(t, argPD, returnedPD)
}

func Test_MetadataProcessor_SilentlyPassesIfDeleteFails(t *testing.T) {
	mockCtrl, mockComputeClient := createMockClient(t)
	defer mockCtrl.Finish()
	mockComputeClient.EXPECT().GetDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(&compute.Disk{}, nil)
	mockComputeClient.EXPECT().CreateDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockComputeClient.EXPECT().DeleteDisk(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("can't find disk"))

	argPD := persistentDisk{uri: "zones/test-zone/disks/disk-name"}
	expectedReturnPD := persistentDisk{uri: "zones/test-zone/disks/disk-name-1"}
	processor := newMetadataProcessor("project", "test-zone", mockComputeClient)
	processor.requiredLicenses = []string{"license/uri"}
	returnedPD, err := processor.process(argPD, nil)

	assert.NoError(t, err)
	assert.Equal(t, expectedReturnPD, returnedPD)
}

func createMockClient(t *testing.T) (*gomock.Controller, *mocks.MockClient) {
	mockCtrl := gomock.NewController(t)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	return mockCtrl, mockComputeClient
}
