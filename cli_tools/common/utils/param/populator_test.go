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

package param

import (
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestPopulator_PopulateMissingParametersReturnsErrorWhenZoneCantBeRetrieved(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := "US"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone("us-west2", project).Return("",
		daisy.Errf("zone not found")).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "zone not found")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndNotRunningOnGCE(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "project cannot be determined because build is not running on GCE")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndGCEProjectIdEmpty(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", nil)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "project cannot be determined")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndMetadataReturnsError(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("pr", daisy.Errf("Err"))
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "project cannot be determined")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenScratchBucketCreationError(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := ""
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().CreateScratchBucket(file, project, zone).Return("", "", daisy.Errf("err"))
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "failed to create scratch bucket")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenScratchBucketInvalidFormat(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "NOT_GCS_PATH"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "invalid scratch bucket")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenPopulateRegionFails(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "NOT_ZONE"
	region := "NOT_REGION"
	file := "gs://a_bucket/a_file"
	storageLocation := "US"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Contains(t, err.Error(), "NOT_ZONE is not a valid zone")
}

func TestPopulator_PopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(t *testing.T) {
	project := "a_project"
	zone := ""
	region := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	storageLocation := "US"

	file := "gs://sourcebucket/sourcefile"
	expectedBucketName := "scratchbucket"
	expectedRegion := "europe-north1"
	expectedZone := "europe-north1-b"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone(expectedRegion, project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs(expectedBucketName).Return(&storage.BucketAttrs{Location: expectedRegion}, nil)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, "europe-north1-b", zone)
	assert.Equal(t, "europe-north1", region)
	assert.Equal(t, "gs://scratchbucket/scratchpath", scratchBucketGcsPath)
}

func TestPopulator_DeleteResources_WhenScratchBucketInAnotherProject(t *testing.T) {
	for _, tt := range []struct {
		caseName             string
		deleteResult         error
		deleteExpected       bool
		expectedError        string
		scratchBucketGCSPath string
		fileGCSPath          string
	}{
		{
			caseName:       "In scratch - Successful deletion",
			deleteResult:   nil,
			deleteExpected: true,
			expectedError: "ScratchBucketNotInProjectError: Scratch bucket \"scratchbucket\" is not in project \"a_project\". " +
				"Deleted \"gs://scratchbucket/sourcefile\"",
			scratchBucketGCSPath: "gs://scratchbucket/scratchpath",
			fileGCSPath:          "gs://scratchbucket/sourcefile",
		},
		{
			caseName:       "In scratch - Failed deletion",
			deleteResult:   errors.New("Failed to delete path"),
			deleteExpected: true,
			expectedError: "ScratchBucketNotInProjectError: Scratch bucket \"scratchbucket\" is not in project \"a_project\". " +
				"Failed to delete \"gs://scratchbucket/sourcefile\". Check with the owner of " +
				"gs://\"scratchbucket\" for more information",
			scratchBucketGCSPath: "gs://scratchbucket/scratchpath",
			fileGCSPath:          "gs://scratchbucket/sourcefile",
		},
		{
			caseName:             "Not in scratch - Don't delete",
			expectedError:        "ScratchBucketNotInProjectError: Scratch bucket \"scratchbucket\" is not in project \"a_project\"",
			scratchBucketGCSPath: "gs://scratchbucket/scratchpath",
			fileGCSPath:          "gs://source-images/sourcefile",
		},
		{
			caseName:             "GCS Image - Don't delete",
			expectedError:        "ScratchBucketNotInProjectError: Scratch bucket \"scratchbucket\" is not in project \"a_project\"",
			scratchBucketGCSPath: "gs://scratchbucket/scratchpath",
			fileGCSPath:          "",
		},
	} {
		t.Run(tt.caseName, func(t *testing.T) {
			project := "a_project"
			zone := ""
			region := ""
			scratchBucketGcsPath := tt.scratchBucketGCSPath
			storageLocation := "US"
			file := tt.fileGCSPath

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
			mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
			mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(false)
			mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
			mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
			if tt.deleteExpected {
				mockStorageClient.EXPECT().DeleteGcsPath(file).Return(tt.deleteResult)
			}

			err := NewPopulator(
				mockMetadataGce,
				mockStorageClient,
				mockResourceLocationRetriever,
				mockScratchBucketCreator,
			).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
				file, &storageLocation)

			assert.EqualError(t, err, tt.expectedError)
		})
	}
}

func TestPopulator_PopulateMissingParametersCreatesScratchBucketIfNotProvided(t *testing.T) {
	project := "a_project"
	zone := ""
	region := ""
	scratchBucketGcsPath := ""
	storageLocation := "US"

	file := "gs://sourcebucket/sourcefile"
	expectedBucketName := "new_scratch_bucket"
	expectedRegion := "europe-north1"
	expectedZone := "europe-north1-c"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)

	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().
		CreateScratchBucket(file, project, zone).
		Return(expectedBucketName, expectedRegion, nil).
		Times(1)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone(expectedRegion, project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, expectedZone, zone)
	assert.Equal(t, expectedRegion, region)
	assert.Equal(t, fmt.Sprintf("gs://%v/", expectedBucketName), scratchBucketGcsPath)
}

func TestPopulator_PopulateMissingParametersCreatesScratchBucketIfNotProvidedOnGCE(t *testing.T) {
	project := "a_project"
	zone := ""
	region := ""
	scratchBucketGcsPath := ""
	storageLocation := "US"

	file := "gs://sourcebucket/sourcefile"
	expectedBucketName := "new_scratch_bucket"
	expectedRegion := "europe-north1"
	expectedZone := "europe-north1-c"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().Zone().Return(expectedZone, nil)

	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().
		CreateScratchBucket(file, project, expectedZone).
		Return(expectedBucketName, expectedRegion, nil).
		Times(1)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone(expectedRegion, project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, expectedZone, zone)
	assert.Equal(t, expectedRegion, region)
	assert.Equal(t, fmt.Sprintf("gs://%v/", expectedBucketName), scratchBucketGcsPath)
}

func TestPopulator_PopulateMissingParametersPopulatesStorageLocationWithScratchBucketLocation(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "us-central1-b"
	region := "us-central1"
	file := "gs://a_bucket/a_file"
	storageLocation := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)
	mockResourceLocationRetriever.EXPECT().GetLargestStorageLocation(region).Return("US")

	err := NewPopulator(
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation)

	assert.Nil(t, err)
	assert.Equal(t, "US", storageLocation)
}
