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
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomainmocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	defaultProject = "project123"
	defaultZone    = "zone1"
)

func TestExecuteRequests_HappyCase(t *testing.T) {
	disk1Name := "disk-1"
	disk2Name := "disk-2"
	dataRequest1 := makeRequest(disk1Name)
	dataRequest2 := makeRequest(disk2Name)

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetDisk(defaultProject, dataRequest1.Zone, disk1Name).Return(nil, errors.New("disk not found"))
	mockCompute.EXPECT().GetDisk(defaultProject, dataRequest2.Zone, disk2Name).Return(nil, errors.New("disk not found"))

	mockSingleImporter := ovfdomainmocks.NewMockDiskImporterInterface(ctrl)
	mockSingleImporter.EXPECT().Import(gomock.Any(), dataRequest1, gomock.Any()).Return(nil)
	mockSingleImporter.EXPECT().Import(gomock.Any(), dataRequest2, gomock.Any()).Return(nil)

	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().NewLogger("[import-disk-1]").Return(logging.NewToolLogger("test"))
	mockLogger.EXPECT().NewLogger("[import-disk-2]").Return(logging.NewToolLogger("test"))

	DiskURIs, actualError := (&requestExecutor{
		computeClient:      mockCompute,
		singleDiskImporter: mockSingleImporter,
		logger:             mockLogger,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		dataRequest1, dataRequest2,
	})
	assert.NoError(t, actualError)

	disk1, err1 := disk.NewDisk(defaultProject, dataRequest1.Zone, disk1Name)
	disk2, err2 := disk.NewDisk(defaultProject, dataRequest2.Zone, disk2Name)
	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.Equal(t, []domain.Disk{disk1, disk2}, DiskURIs)
}

func TestExecuteRequests_DontImport_IfTheFirstDiskIsAlreadyExists(t *testing.T) {
	disk1Name := "disk-1"
	disk2Name := "disk-2"
	dataRequest1 := makeRequest(disk1Name)
	dataRequest2 := makeRequest(disk2Name)

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetDisk(defaultProject, dataRequest1.Zone, disk1Name).Return(nil, nil)

	_, actualError := (&requestExecutor{
		computeClient: mockCompute,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		dataRequest1, dataRequest2,
	})
	assert.EqualError(t, actualError, fmt.Sprintf("Intermediate disk %s already exists. Re-run create-disk.", disk1Name))
}

func TestExecuteRequests_DontImport_IfTheSecondDiskIsAlreadyExists(t *testing.T) {
	disk1Name := "disk-1"
	disk2Name := "disk-2"
	dataRequest1 := makeRequest(disk1Name)
	dataRequest2 := makeRequest(disk2Name)

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetDisk(defaultProject, dataRequest1.Zone, disk1Name).Return(nil, errors.New("disk not found"))
	mockCompute.EXPECT().GetDisk(defaultProject, dataRequest2.Zone, disk2Name).Return(nil, nil)

	_, actualError := (&requestExecutor{
		computeClient: mockCompute,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		dataRequest1, dataRequest2,
	})
	assert.EqualError(t, actualError, fmt.Sprintf("Intermediate disk %s already exists. Re-run create-disk.", disk2Name))
}

func TestExecuteRequests_ReturnError_WhenTimeoutExceeded(t *testing.T) {
	disk1Name := "disk-1"
	request := makeRequest(disk1Name)

	request.Timeout = 0
	_, actualError := (&requestExecutor{}).executeRequests(context.Background(), []importer.ImageImportRequest{request})
	assert.EqualError(t, actualError, "Timeout exceeded")
}

func makeRequest(diskName string) importer.ImageImportRequest {
	return importer.ImageImportRequest{
		ExecutionID:        diskName,
		Project:            defaultProject,
		DaisyLogLinePrefix: diskName,
		Timeout:            time.Hour,
		Zone:               defaultZone,
		DiskName:           diskName,
	}
}
