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
	mockSingleImporter.EXPECT().ImportImage(gomock.Any(), bootRequest, gomock.Any()).Return(nil)
	mockSingleImporter.EXPECT().ImportImage(gomock.Any(), dataRequest, gomock.Any()).Return(nil)

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
	assert.Equal(t, imageURIs, []string{"projects/project123/global/images/img-1", "projects/project123/global/images/img-2"})
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

func TestExecuteRequests_PerformCleanup_IfAnyImportFails(t *testing.T) {
	importFailedError := errors.New("Error failed")
	bootRequest := makeRequest("img-1")
	dataRequest := makeRequest("img-2")

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	// Prior to import, no images exist.
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, errors.New("image not found"))
	mockCompute.EXPECT().GetImage(defaultProject, dataRequest.ImageName).Return(nil, errors.New("image not found"))

	// Both images are created.
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, nil)
	mockCompute.EXPECT().GetImage(defaultProject, dataRequest.ImageName).Return(nil, nil)

	// Expect they're both deleted.
	mockCompute.EXPECT().DeleteImage(defaultProject, bootRequest.ImageName).Return(nil)
	mockCompute.EXPECT().DeleteImage(defaultProject, dataRequest.ImageName).Return(nil)

	mockSingleImporter := ovfdomainmocks.NewMockImageImporterInterface(ctrl)
	mockSingleImporter.EXPECT().ImportImage(gomock.Any(), bootRequest, gomock.Any()).Return(nil)
	mockSingleImporter.EXPECT().ImportImage(gomock.Any(), dataRequest, gomock.Any()).Return(importFailedError)

	_, actualError := (&requestExecutor{
		computeClient:  mockCompute,
		singleImporter: mockSingleImporter,
		logger:         logging.NewToolLogger("test"),
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		bootRequest, dataRequest,
	})
	assert.Equal(t, importFailedError, actualError)
}

func TestExecuteRequests_ReturnError_WhenTimeoutExceeded(t *testing.T) {
	request := makeRequest("img-1")
	request.Timeout = 0
	_, actualError := (&requestExecutor{}).executeRequests(context.Background(), []importer.ImageImportRequest{request})
	assert.EqualError(t, actualError, "Timeout exceeded")
}

func TestExecuteRequests_LogsMessage_IfCleanupFails(t *testing.T) {
	importFailedError := errors.New("Error failed")
	bootRequest := makeRequest("img-1")

	ctrl := gomock.NewController(t)
	mockCompute := mocks.NewMockClient(ctrl)
	// Prior to import, the image doesn't exist.
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, errors.New("image not found"))

	// Image is created during import.
	mockCompute.EXPECT().GetImage(defaultProject, bootRequest.ImageName).Return(nil, nil)

	// Image fails to delete
	mockCompute.EXPECT().DeleteImage(defaultProject, bootRequest.ImageName).Return(errors.New("Failed to delete disk"))

	mockSingleImporter := ovfdomainmocks.NewMockImageImporterInterface(ctrl)
	mockSingleImporter.EXPECT().ImportImage(gomock.Any(), bootRequest, gomock.Any()).Return(importFailedError)

	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().NewLogger("[import-img-1]").Return(logging.NewToolLogger("test"))
	mockLogger.EXPECT().User("Failed to delete \"projects/project123/global/images/img-1\". Manual deletion required.")
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()

	_, actualError := (&requestExecutor{
		computeClient:  mockCompute,
		singleImporter: mockSingleImporter,
		logger:         mockLogger,
	}).executeRequests(context.Background(), []importer.ImageImportRequest{
		bootRequest,
	})
	assert.Equal(t, importFailedError, actualError)
}

func makeRequest(imageName string) importer.ImageImportRequest {
	return importer.ImageImportRequest{
		ImageName:          imageName,
		Project:            defaultProject,
		DaisyLogLinePrefix: imageName,
		Timeout:            time.Hour,
	}
}
