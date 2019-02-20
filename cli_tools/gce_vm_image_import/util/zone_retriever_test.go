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

package gcevmimageimportutil

import (
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
	"testing"
)

func TestGetZoneFromGCEMetadata(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenGCEMetadataReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("", fmt.Errorf("err"))
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneErrorWhenGCEMetadataReturnsEmtpyZone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("", nil)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneErrorWhenNotOnGCEAndNoStorageRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneFromStorageRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createZone("us-west1", "b"),
		createZone("us-west2", "a"),
		createZone("us-west2", "b"),
		createZone("us-west2", "c"),
		createZone("europe-north1", "c"),
		createZone("europe-north2", "a"),
		createZone("europe-north2", "b"),
		createZone("europe-north2", "c"),
		createZone("europe-west3", "a"),
		createZone("europe-west3", "b"),
	}
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)
	mockComputeService.EXPECT().GetZones(projectID).Return(zones, nil)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north2-a", zone)
}

func TestGetZoneFromGCEWhenNoMatchingZone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createZone("us-west1", "b"),
		createZone("us-west2", "a"),
		createZone("us-west2", "b"),
		createZone("us-west2", "c"),
	}
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)
	mockComputeService.EXPECT().GetZones(projectID).Return(zones, nil)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenNoMatchingZoneAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createZone("us-west1", "b"),
		createZone("us-west2", "a"),
		createZone("us-west2", "b"),
		createZone("us-west2", "c"),
	}
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)
	mockComputeService.EXPECT().GetZones(projectID).Return(zones, nil)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneFromGCEWhenGetComputeServiceReturnsZoneError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)
	mockComputeService.EXPECT().GetZones(projectID).Return(nil, fmt.Errorf("zone error"))

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenGetComputeServiceReturnsZoneErrorAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)
	mockComputeService.EXPECT().GetZones(projectID).Return(nil, fmt.Errorf("zone error"))

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneFromGCEWhenProjectNotSetAndStorageRegionSet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", "")

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenProjectNotSetAndStorageRegionSetAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockComputeServiceInterface(mockCtrl)

	zr := ZoneRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("EUROPE-NORTH2", "")

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func createZone(region string, zoneSuffix string) *compute.Zone {
	return &compute.Zone{Name: region + "-" + zoneSuffix, Region: "/regions/" + region}
}
