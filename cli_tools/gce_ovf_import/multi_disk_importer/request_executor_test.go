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
	var requests []importer.ImageImportRequest
	var dataDisks []domain.Disk
	for i := 0; i < 2; i++ {
		imgName := fmt.Sprintf("img-name-%d", i+1)
		requests = append(requests, makeRequest(imgName))
		diskName := fmt.Sprintf("disk-%s", requests[i].ExecutionID)
		dataDisk, err := disk.NewDisk(requests[i].Project, requests[i].Zone, diskName)

		assert.NoError(t, err)

		dataDisks = append(dataDisks, dataDisk)
	}

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)

	mockSingleImporter := ovfdomainmocks.NewMockDiskImporterInterface(ctrl)
	mockSingleImporter.EXPECT().Import(gomock.Any(), requests[0], gomock.Any()).Return(dataDisks[0].GetURI(), nil)
	mockSingleImporter.EXPECT().Import(gomock.Any(), requests[1], gomock.Any()).Return(dataDisks[1].GetURI(), nil)

	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().NewLogger("[import-disk-1]").Return(logging.NewToolLogger("test"))
	mockLogger.EXPECT().NewLogger("[import-disk-2]").Return(logging.NewToolLogger("test"))

	actualDisks, actualError := (&requestExecutor{
		computeClient:      mockCompute,
		singleDiskImporter: mockSingleImporter,
		logger:             mockLogger,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		requests[0], requests[1],
	})
	assert.NoError(t, actualError)

	assert.Equal(t, len(dataDisks), len(actualDisks))
	assert.Equal(t, dataDisks, actualDisks)
}

func makeRequest(imgName string) importer.ImageImportRequest {
	return importer.ImageImportRequest{
		ExecutionID:        imgName,
		Project:            defaultProject,
		DaisyLogLinePrefix: imgName,
		Timeout:            time.Hour,
		Zone:               defaultZone,
		ImageName:          imgName,
	}
}
