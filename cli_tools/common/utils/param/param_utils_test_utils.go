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

package paramutils

import (
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
)

// RunTestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion is a test helper function
func RunTestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(
		t *testing.T, zone *string, region *string, scratchBucketGcsPath *string, file *string,
		project *string, expectedBucketName string, expectedRegion string, expectedZone string) error {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone(expectedRegion, *project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs(expectedBucketName).Return(&storage.BucketAttrs{Location: expectedRegion}, nil)

	return PopulateMissingParameters(project, zone, region, scratchBucketGcsPath,
		*file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)
}

// RunTestPopulateMissingParametersCreatesScratchBucketIfNotProvided is a test helper function
func RunTestPopulateMissingParametersCreatesScratchBucketIfNotProvided(
		t *testing.T, zone *string, region *string, scratchBucketGcsPath *string, file *string,
		project *string, expectedProject string, expectedBucket string, expectedRegion string, expectedZone string) error {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)

	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().
		CreateScratchBucket(*file, *project).
		Return(expectedBucket, expectedRegion, nil).
		Times(1)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone(expectedRegion, expectedProject).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	return PopulateMissingParameters(project, zone, region, scratchBucketGcsPath,
		*file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)
}

// RunTestPopulateProjectIfMissingProjectPopulatedFromGCE is a test helper function
func RunTestPopulateProjectIfMissingProjectPopulatedFromGCE(t *testing.T, project *string, expectedProject string) error {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return(expectedProject, nil)

	return PopulateProjectIfMissing(mockMetadataGce, project)
}