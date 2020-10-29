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

package ovfexporter

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestExporter_Export(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	params := ovfexportdomain.GetAllInstanceExportParams()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockLogger := mocks.NewMockLoggerInterface(mockCtrl)
	mockInspector := mocks.NewMockInspector(mockCtrl)
	mockOvfDescriptorGenerator := mocks.NewMockOvfDescriptorGenerator(mockCtrl)
	mockOvfManifestGenerator := mocks.NewMockOvfManifestGenerator(mockCtrl)
	mockInstanceDisksExporter := mocks.NewMockInstanceDisksExporter(mockCtrl)
	mockInstanceExportPreparer := mocks.NewMockInstanceExportPreparer(mockCtrl)
	mockInstanceExportCleaner := mocks.NewMockInstanceExportCleaner(mockCtrl)

	exporter := &OVFExporter{
		storageClient:          mockStorageClient,
		computeClient:          mockComputeClient,
		mgce:                   mockMetadataGce,
		bucketIteratorCreator:  mockBucketIteratorCreator,
		Logger:                 mockLogger,
		params:                 params,
		loggableBuilder:        service.NewOVFExportLoggableBuilder(),
		ovfDescriptorGenerator: mockOvfDescriptorGenerator,
		manifestFileGenerator:  mockOvfManifestGenerator,
		inspector:              mockInspector,
		instanceDisksExporter:  mockInstanceDisksExporter,
		instanceExportPreparer: mockInstanceExportPreparer,
		instanceExportCleaner:  mockInstanceExportCleaner,
	}
	err := exporter.run(context.Background())
	assert.Nil(t, err)
}
