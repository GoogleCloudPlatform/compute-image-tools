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

package param

import (
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/paramhelper"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
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
		got, err := paramhelper.GetRegion(*zone)
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
	storageLocation := "US"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone("us-west2", project).Return("", daisy.Errf("err")).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndNotRunningOnGCE(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndGCEProjectIdEmpty(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndMetadataReturnsError(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketCreationError(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenScratchBucketInvalidFormat(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersReturnsErrorWhenPopulateRegionFails(t *testing.T) {
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
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.NotNil(t, err)
}

func TestPopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(t *testing.T) {
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
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockResourceLocationRetriever.EXPECT().GetZone(expectedRegion, project).Return(expectedZone, nil).Times(1)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs(expectedBucketName).Return(&storage.BucketAttrs{Location: expectedRegion}, nil)

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath, file, &storageLocation,
		mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, "europe-north1-b", zone)
	assert.Equal(t, "europe-north1", region)
	assert.Equal(t, "gs://scratchbucket/scratchpath", scratchBucketGcsPath)
}

func TestPopulateMissingParametersCreatesScratchBucketIfNotProvided(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath, file, &storageLocation,
		mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, expectedZone, zone)
	assert.Equal(t, expectedRegion, region)
	assert.Equal(t, fmt.Sprintf("gs://%v/", expectedBucketName), scratchBucketGcsPath)
}

func TestPopulateMissingParametersCreatesScratchBucketIfNotProvidedOnGCE(t *testing.T) {
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

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath, file, &storageLocation,
		mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, expectedZone, zone)
	assert.Equal(t, expectedRegion, region)
	assert.Equal(t, fmt.Sprintf("gs://%v/", expectedBucketName), scratchBucketGcsPath)
}

func TestPopulateProjectIfMissingProjectPopulatedFromGCE(t *testing.T) {
	project := ""
	expectedProject := "gce_project"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return(expectedProject, nil)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.Nil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfMissingProjectNotOnGCE(t *testing.T) {
	project := ""
	expectedProject := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.NotNil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfNotMissingProject(t *testing.T) {
	project := "aProject"
	expectedProject := "aProject"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.Nil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateProjectIfMissingProjectWithErrorRetrievingFromGCE(t *testing.T) {
	project := ""
	expectedProject := ""

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", daisy.Errf("gce error"))

	err := PopulateProjectIfMissing(mockMetadataGce, &project)

	assert.NotNil(t, err)
	assert.Equal(t, expectedProject, project)
}

func TestPopulateMissingParametersPopulatesStorageLocationWithScratchBucketLocation(t *testing.T) {
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
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)
	mockResourceLocationRetriever.EXPECT().GetLargestStorageLocation(region).Return("US")

	err := PopulateMissingParameters(&project, &zone, &region, &scratchBucketGcsPath,
		file, &storageLocation, mockMetadataGce, mockScratchBucketCreator, mockResourceLocationRetriever, mockStorageClient)

	assert.Nil(t, err)
	assert.Equal(t, "US", storageLocation)
}

func TestGetGlobalResourcePathFromNameOnly(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "aNetwork")
	assert.Equal(t, "global/networks/aNetwork", n)
}

func TestGetGlobalResourcePathFromRelativeURL(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetGlobalResourcePathFromFullURL(t *testing.T) {
	var n = GetGlobalResourcePath("networks", "https://www.googleapis.com/compute/v1/x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetRegionalResourcePathFromNameOnly(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "aSubnetwork")
	assert.Equal(t, "regions/aRegion/subnetworks/aSubnetwork", n)
}

func TestGetRegionalResourcePathFromRelativeURL(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "x/blabla")
	assert.Equal(t, "x/blabla", n)
}

func TestGetRegionalResourcePathFromFullURL(t *testing.T) {
	var n = GetRegionalResourcePath("aRegion", "subnetworks", "https://www.googleapis.com/compute/v1/x/blabla")
	assert.Equal(t, "x/blabla", n)
}
