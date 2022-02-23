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

package storage

import (
	"fmt"
	"testing"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

var (
	defaultZones = []*compute.Zone{
		createUpZone("us-west1", "b"),
		createUpZone("us-west2", "a"),
		createUpZone("us-west2", "b"),
		createUpZone("us-west2", "c"),
		createUpZone("us-central2", "a"),
		createUpZone("us-central2", "b"),
		createUpZone("us-central1", "a"),
		createUpZone("us-central1", "b"),
		createUpZone("europe-north1", "c"),
		createUpZone("europe-north2", "a"),
	}
)

func TestGetZoneFromGCEMetadata(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenGCEMetadataReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("", daisy.Errf("err"))
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("", projectID)

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
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneErrorWhenNotOnGCEAndNoStorageRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneFromStorageRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createUpZone("us-west1", "b"),
		createUpZone("us-west2", "a"),
		createUpZone("us-west2", "b"),
		createUpZone("us-west2", "c"),
		createUpZone("europe-north1", "c"),
		createUpZone("europe-north2", "a"),
		createUpZone("europe-north2", "b"),
		createUpZone("europe-north2", "c"),
		createUpZone("europe-west3", "a"),
		createUpZone("europe-west3", "b"),
	}
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(zones, nil)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north2-a", zone)
}

func TestGetZoneFromGCEWhenNoMatchingZone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createUpZone("us-west1", "b"),
		createUpZone("us-west2", "a"),
		createUpZone("us-west2", "b"),
		createUpZone("us-west2", "c"),
	}
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(zones, nil)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneFromStorageMultiRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(defaultZones, nil)

	zr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := zr.GetZone("US", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "us-central1-a", zone)
}

func TestGetZoneFromGCEWhenMultiRegionHasNoValidZones(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)

	zones := []*compute.Zone{
		createUpZone("us-west1", "b"),
		createUpZone("us-west2", "a"),
		createUpZone("us-west2", "b"),
		createUpZone("us-west2", "c"),
	}
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(zones, nil)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("ASIA", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneFromGCEWhenMultiRegionHasNoZonesUP(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	zones := []*compute.Zone{
		createDownZone("us-west1", "b"),
		createDownZone("us-west2", "a"),
		createDownZone("us-west2", "b"),
		createDownZone("us-west2", "c"),
		createDownZone("us-central2", "a"),
		createUpZone("europe-north2", "b"),
		createUpZone("europe-north2", "c"),
		createUpZone("europe-west3", "a"),
	}

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("asia-east1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)

	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(zones, nil)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("US", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "asia-east1-c", zone)
}

func TestGetZoneErrorWhenNoMatchingZoneAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(defaultZones, nil)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("ASIA-EAST1", projectID)

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
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(nil, fmt.Errorf("zone error"))

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", projectID)

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenGetComputeServiceReturnsZoneErrorAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	projectID := "a_project"
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	mockComputeService.EXPECT().ListZones(projectID).Return(nil, fmt.Errorf("zone error"))

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", projectID)

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func TestGetZoneFromGCEWhenProjectNotSetAndStorageRegionSet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().Zone().Return("europe-north1-c", nil).Times(1)
	mockMetadataGce.EXPECT().OnGCE().Return(true).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", "")

	assert.Nil(t, err)
	assert.Equal(t, "europe-north1-c", zone)
}

func TestGetZoneErrorWhenProjectNotSetAndStorageRegionSetAndNotOnGCE(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false).Times(1)
	mockComputeService := mocks.NewMockClient(mockCtrl)

	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}
	zone, err := rlr.GetZone("EUROPE-NORTH2", "")

	assert.NotNil(t, err)
	assert.Equal(t, "", zone)
}

func createUpZone(region string, zoneSuffix string) *compute.Zone {
	return createZone(region, zoneSuffix, "UP")
}

func createDownZone(region string, zoneSuffix string) *compute.Zone {
	return createZone(region, zoneSuffix, "DOWN")
}

func createZone(region string, zoneSuffix string, status string) *compute.Zone {
	return &compute.Zone{Name: region + "-" + zoneSuffix, Region: "/regions/" + region, Status: status}
}

func TestGetLargestStorageLocationForMultiRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}

	location := rlr.GetLargestStorageLocation("US")
	assert.Equal(t, "US", location)
}

func TestGetLargestStorageLocationForRegionInMultiRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}

	location := rlr.GetLargestStorageLocation("europe-west2")
	assert.Equal(t, "EU", location)
}

func TestGetLargestStorageLocationForRegionAllCapsInMultiRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}

	location := rlr.GetLargestStorageLocation("EUROPE-WEST2")
	assert.Equal(t, "EU", location)
}

func TestGetLargestStorageLocationForRegionNotInMultiRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}

	location := rlr.GetLargestStorageLocation("australia-southeast1")
	assert.Equal(t, "australia-southeast1", location)
}

func TestGetLargestStorageLocationForMultiRegionNoDualRegions(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockComputeService := mocks.NewMockClient(mockCtrl)
	rlr := ResourceLocationRetriever{mockMetadataGce, mockComputeService}

	location := rlr.GetLargestStorageLocation("us-central1")
	// We don't want NAM4 here
	assert.Equal(t, "US", location)
}
