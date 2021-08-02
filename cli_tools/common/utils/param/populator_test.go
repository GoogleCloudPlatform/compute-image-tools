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
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func TestPopulator_PopulateMissingParametersReturnsErrorWhenZoneCantBeRetrieved(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

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
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "zone not found")
}

func TestPopulator_PropagatesErrorFromNetworkResolver(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "zone"
	region := "region"
	file := "gs://a_bucket/a_file"
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)
	mockNetworkResolver := mocks.NewMockNetworkResolver(mockCtrl)
	mockNetworkResolver.EXPECT().ResolveAndValidateNetworkAndSubnet("original-network", "original-subnet", "region", "a_project").Return("", "", daisy.Errf("network cannot be found"))
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "network cannot be found")
}

func TestPopulator_UsesReturnValuesFromNetworkResolver(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "us-west2-a"
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: "us-west2"}, nil).Times(1)
	mockNetworkResolver := mocks.NewMockNetworkResolver(mockCtrl)
	mockNetworkResolver.EXPECT().ResolveAndValidateNetworkAndSubnet(
		"original-network", "original-subnet", "us-west2", "a_project").Return("fixed-network", "fixed-subnet", nil)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)
	assert.NoError(t, err)
	assert.Equal(t, "fixed-network", network)
	assert.Equal(t, "fixed-subnet", subnet)
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndNotRunningOnGCE(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "project cannot be determined because build is not running on GCE")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndGCEProjectIdEmpty(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("", nil)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "project cannot be determined")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenProjectNotProvidedAndMetadataReturnsError(t *testing.T) {
	project := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(true)
	mockMetadataGce.EXPECT().ProjectID().Return("pr", daisy.Errf("Err"))
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "project cannot be determined")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenScratchBucketCreationError(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := ""
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockMetadataGce.EXPECT().OnGCE().Return(false)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().CreateScratchBucket(file, project, zone).Return("", "", daisy.Errf("err"))
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "failed to create scratch bucket")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenScratchBucketInvalidFormat(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "NOT_GCS_PATH"
	zone := ""
	region := ""
	file := "gs://a_bucket/a_file"
	storageLocation := ""
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "invalid scratch bucket")
}

func TestPopulator_PopulateMissingParametersReturnsErrorWhenPopulateRegionFails(t *testing.T) {
	project := "a_project"
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	zone := "NOT_ZONE"
	region := "NOT_REGION"
	file := "gs://a_bucket/a_file"
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Contains(t, err.Error(), "NOT_ZONE is not a valid zone")
}

func TestPopulator_PopulateMissingParametersDoesNotChangeProvidedScratchBucketAndUsesItsRegion(t *testing.T) {
	project := "a_project"
	zone := ""
	region := ""
	scratchBucketGcsPath := "gs://scratchbucket/scratchpath"
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

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
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Nil(t, err)
	assert.Equal(t, "a_project", project)
	assert.Equal(t, "europe-north1-b", zone)
	assert.Equal(t, "europe-north1", region)
	assert.Equal(t, "gs://scratchbucket/scratchpath", scratchBucketGcsPath)
}

func TestPopulator_DeleteResources_WhenScratchBucketInAnotherProject(t *testing.T) {
	for _, tt := range []struct {
		caseName                string
		client                  string
		deleteResult            error
		deleteExpected          bool
		expectedError           string
		expectedAnonymizedError string
		scratchBucketGCSPath    string
		fileGCSPath             string
	}{
		{
			caseName:       "In scratch - gcloud - Successful deletion",
			client:         "gcloud",
			deleteResult:   nil,
			deleteExpected: true,
			expectedError: "Scratch bucket \"scratchbucket\" is not in project \"a_project\". " +
				"Deleted \"gs://scratchbucket/sourcefile\"",
			expectedAnonymizedError: "Scratch bucket %q is not in project %q. Deleted %q",
			scratchBucketGCSPath:    "gs://scratchbucket/scratchpath",
			fileGCSPath:             "gs://scratchbucket/sourcefile",
		},
		{
			caseName:       "In scratch - gcloud - Failed deletion",
			client:         "gcloud",
			deleteResult:   errors.New("Failed to delete path"),
			deleteExpected: true,
			expectedError: "Scratch bucket \"scratchbucket\" is not in project \"a_project\". Failed to delete " +
				"\"gs://scratchbucket/sourcefile\": Failed to delete path. " +
				"Check with the owner of gs://\"scratchbucket\" for more information",
			expectedAnonymizedError: "Scratch bucket %q is not in project %q. Failed to delete %q: %v. " +
				"Check with the owner of gs://%q for more information",
			scratchBucketGCSPath: "gs://scratchbucket/scratchpath",
			fileGCSPath:          "gs://scratchbucket/sourcefile",
		},
		{
			caseName:                "In scratch - not gcloud - don't delete",
			client:                  "api",
			expectedError:           "Scratch bucket \"scratchbucket\" is not in project \"a_project\"",
			expectedAnonymizedError: "Scratch bucket %q is not in project %q",
			scratchBucketGCSPath:    "gs://scratchbucket/scratchpath",
			fileGCSPath:             "gs://scratchbucket/sourcefile",
		},
		{
			caseName:                "Not in scratch - Don't delete",
			client:                  "gcloud",
			expectedError:           "Scratch bucket \"scratchbucket\" is not in project \"a_project\"",
			expectedAnonymizedError: "Scratch bucket %q is not in project %q",
			scratchBucketGCSPath:    "gs://scratchbucket/scratchpath",
			fileGCSPath:             "gs://source-images/sourcefile",
		},
		{
			caseName:                "GCS Image - Don't delete",
			client:                  "gcloud",
			expectedError:           "Scratch bucket \"scratchbucket\" is not in project \"a_project\"",
			expectedAnonymizedError: "Scratch bucket %q is not in project %q",
			scratchBucketGCSPath:    "gs://scratchbucket/scratchpath",
			fileGCSPath:             "",
		},
	} {
		t.Run(tt.caseName, func(t *testing.T) {
			project := "a_project"
			zone := ""
			region := ""
			scratchBucketGcsPath := tt.scratchBucketGCSPath
			storageLocation := "US"
			file := tt.fileGCSPath
			network := "original-network"
			subnet := "original-subnet"

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
			mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
			mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(false)
			mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
			mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
			if tt.deleteExpected {
				mockStorageClient.EXPECT().DeleteObject(file).Return(tt.deleteResult)
			}
			mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
			err := NewPopulator(
				mockNetworkResolver,
				mockMetadataGce,
				mockStorageClient,
				mockResourceLocationRetriever,
				mockScratchBucketCreator,
			).PopulateMissingParameters(&project, tt.client, &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

			realError := err.(daisy.DError)
			assert.EqualError(t, realError, tt.expectedError)
			assert.Equal(t, strings.Join(realError.AnonymizedErrs(), ""), tt.expectedAnonymizedError)
		})
	}
}

func TestPopulator_PopulateMissingParametersCreatesScratchBucketIfNotProvided(t *testing.T) {
	project := "a_project"
	zone := ""
	region := ""
	scratchBucketGcsPath := ""
	storageLocation := "US"
	network := "original-network"
	subnet := "original-subnet"

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
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

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
	network := "original-network"
	subnet := "original-subnet"

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
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

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
	network := "original-network"
	subnet := "original-subnet"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMetadataGce := mocks.NewMockMetadataGCEInterface(mockCtrl)
	mockScratchBucketCreator := mocks.NewMockScratchBucketCreatorInterface(mockCtrl)
	mockScratchBucketCreator.EXPECT().IsBucketInProject(project, "scratchbucket").Return(true)
	mockResourceLocationRetriever := mocks.NewMockResourceLocationRetrieverInterface(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("scratchbucket").Return(&storage.BucketAttrs{Location: region}, nil)
	mockResourceLocationRetriever.EXPECT().GetLargestStorageLocation(region).Return("US")
	mockNetworkResolver := newNoOpNetworkResolver(mockCtrl)
	err := NewPopulator(
		mockNetworkResolver,
		mockMetadataGce,
		mockStorageClient,
		mockResourceLocationRetriever,
		mockScratchBucketCreator,
	).PopulateMissingParameters(&project, "gcloud", &zone, &region, &scratchBucketGcsPath, file, &storageLocation, &network, &subnet)

	assert.Nil(t, err)
	assert.Equal(t, "US", storageLocation)
}

func newNoOpNetworkResolver(ctrl *gomock.Controller) NetworkResolver {
	m := mocks.NewMockNetworkResolver(ctrl)
	m.EXPECT().ResolveAndValidateNetworkAndSubnet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	return m
}
