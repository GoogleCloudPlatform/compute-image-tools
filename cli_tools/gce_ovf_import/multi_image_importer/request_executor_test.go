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

package multiimageimporter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image/importer"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomainmocks "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

const (
	defaultProject = "project123"
)

func TestExecuteRequests_HappyCase(t *testing.T) {
	bootRequest := makeRequest("img-1")
	dataRequest := makeRequest("img-2")

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, errors.New("image not found"))
	mockCompute.EXPECT().GetImage(defaultProject, dataRequest.ImageName).Return(nil, errors.New("image not found"))

	mockSingleImporter := ovfdomainmocks.NewMockImageImporterInterface(ctrl)
	mockSingleImporter.EXPECT().Import(gomock.Any(), bootRequest, gomock.Any()).Return(nil)
	mockSingleImporter.EXPECT().Import(gomock.Any(), dataRequest, gomock.Any()).Return(nil)

	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().NewLogger("[import-img-1]").Return(logging.NewToolLogger("test"))
	mockLogger.EXPECT().NewLogger("[import-img-2]").Return(logging.NewToolLogger("test"))

	imageURIs, actualError := (&requestExecutor{
		computeClient:  mockCompute,
		singleImporter: mockSingleImporter,
		logger:         mockLogger,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		bootRequest, dataRequest,
	})
	assert.NoError(t, actualError)
	assert.Equal(t, []domain.Image{
		image.NewImage("project123", "img-1"),
		image.NewImage("project123", "img-2")},
		imageURIs)
}

func TestExecuteRequests_DontImport_IfBootImageAlreadyExists(t *testing.T) {
	bootRequest := makeRequest("img-1")
	dataRequest := makeRequest("img-2")

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, nil)

	_, actualError := (&requestExecutor{
		computeClient: mockCompute,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		bootRequest, dataRequest,
	})
	assert.EqualError(t, actualError, "Intermediate image img-1 already exists. Re-run import.")
}

func TestExecuteRequests_DontImport_IfDataImageAlreadyExists(t *testing.T) {
	bootRequest := makeRequest("img-1")
	dataRequest := makeRequest("img-2")

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, errors.New("image not found"))
	mockCompute.EXPECT().GetImage(defaultProject, dataRequest.ImageName).Return(nil, nil)

	_, actualError := (&requestExecutor{
		computeClient: mockCompute,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		bootRequest, dataRequest,
	})
	assert.EqualError(t, actualError, "Intermediate image img-2 already exists. Re-run import.")
}

func TestExecuteRequests_ReturnError_WhenTimeoutExceeded(t *testing.T) {
	request := makeRequest("img-1")
	request.Timeout = 0
	_, actualError := (&requestExecutor{}).executeRequests(context.Background(), []importer.ImageImportRequest{request})
	assert.EqualError(t, actualError, "Timeout exceeded")
}

func makeRequest(imageName string) importer.ImageImportRequest {
	return importer.ImageImportRequest{
		ImageName:          imageName,
		Project:            defaultProject,
		DaisyLogLinePrefix: imageName,
		Timeout:            time.Hour,
	}
}
