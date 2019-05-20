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
	"cloud.google.com/go/storage"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe-north1-a", "europe-north1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	for _, test := range tests {
		zone := &test.input
		got, err := GetRegion(*zone)
		if test.want != got {
			t.Errorf("%v != %v", test.want, got)
		} else if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		}
	}
}

func TestPopulateRegion(t *testing.T) {
	tests := []struct {
		input string
		want  string
		err   error
	}{
		{"us-central1-c", "us-central1", nil},
		{"europe", "", fmt.Errorf("%v is not a valid zone", "europe")},
		{"", "", fmt.Errorf("zone is empty. Can't determine region")},
	}

	for _, test := range tests {
		zone := &test.input
		regionInit := ""
		region := &regionInit
		err := PopulateRegion(region, *zone)
		if err != test.err && test.err.Error() != err.Error() {
			t.Errorf("%v != %v", test.err, err)
		} else if region != nil && test.want != *region {
			t.Errorf("%v != %v", test.want, *region)
		}
	}
}

func TestPopulateMissingParametersReturnsErrorWhenZoneCantBeRetrieved(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockZoneRetriever.EXPECT().GetZone("us-west2", project).Return("", fmt.Errorf("err")).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndNotRunningOnGCE(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndGCEProjectIdEmpty(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", nil)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndMetadataReturnsError(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("pr", fmt.Errorf("Err"))
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketCreationError(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := ""
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().CreateScratchBucket(file, project).Return("", "", fmt.Errorf("err"))
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketInvalidFormat(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "NOT_GCS_PATH"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenPopulateRegionFails(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "NOT_ZONE"
	region := "NOT_REGION"
	file := "gs://a_bucket/a_file"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockZoneRetriever := mocks.NewMockZoneRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, mockMetadataGce, mockScratchBucketCreator, mockZoneRetriever, mockStorageClient)

	assert.NotNil(t, err)
}
