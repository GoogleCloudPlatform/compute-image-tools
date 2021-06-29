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

package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
)

func TestCreateScratchBucketErrorWhenProjectNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("", "", "")
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func TestCreateScratchBucketNoSourceFileDefaultBucketCreatedBasedOnDefaultRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	expectedBucket := "project1-daisy-bkt-us"
	expectedRegion := "US"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("", project, "")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketNoSourceFileTranslateGoogleDomainDefaultBucketCreatedBasedOnDefaultRegion(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "google.com:proJect1"
	expectedBucket := "elgoog_com-project1-daisy-bkt-us"
	expectedRegion := "US"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("", project, "")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketNoSourceFileBucketCreatedBasedOnInputZone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	expectedBucket := "project1-daisy-bkt-asia-east1"
	expectedRegion := "asia-east1"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     "asia-east1",
		StorageClass: regionalStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("", project, "asia-east1-b")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketNoSourceErrorCreatingDefaultBucket(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	wouldBeBucketName := "project1-daisy-bkt-us"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().CreateBucket(wouldBeBucketName, project, &storage.BucketAttrs{
		Name:         wouldBeBucketName,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(fmt.Errorf("some error"))

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("", project, "")
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", region)
	assert.NotNil(t, err)
	assert.Equal(t, "some error", err.Error())
}

func TestCreateScratchBucketNewBucketCreatedProject(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	project := "PROJECT1"

	ctx := context.Background()

	sourceBucketAttrs := &storage.BucketAttrs{
		Name:         "sourcebucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	anotherBucketAttrs := &storage.BucketAttrs{
		Name:         "anotherbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}

	scratchBucketAttrs := &storage.BucketAttrs{
		Name:         "project1-daisy-bkt-us-west2",
		Location:     "us-west2",
		StorageClass: "regional",
	}

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs(sourceBucketAttrs.Name).Return(sourceBucketAttrs, nil).Times(1)
	mockStorageClient.EXPECT().CreateBucket("project1-daisy-bkt-us-west2", project, scratchBucketAttrs).Return(nil).Times(1)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(sourceBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	gomock.InOrder(
		first,
		second,
		third,
	)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, project).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project, "")
	assert.Equal(t, "project1-daisy-bkt-us-west2", bucket)
	assert.Equal(t, "us-west2", region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketInvalidSourceFileErrorThrown(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"

	c := ScratchBucketCreator{}
	gcsPath := "NOT_A_GS_PATH"
	_, _, err := c.CreateScratchBucket(gcsPath, project, "")
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), fmt.Sprintf("file GCS path `%v` is invalid:", gcsPath)))
}

func TestCreateScratchBucketErrorRetrievingSourceFileBucketMetadataDefaultBucketCreated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	expectedBucket := "project1-daisy-bkt-us"
	expectedRegion := "US"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(&storage.BucketAttrs{}, fmt.Errorf("error retrieving bucket attrs")).Times(1)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project, "")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketErrorRetrievingSourceFileBucketMetadataBucketCreatedBasedOnInputZone(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	expectedBucket := "project1-daisy-bkt-asia-east1"
	expectedRegion := "asia-east1"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(&storage.BucketAttrs{}, fmt.Errorf("error retrieving bucket attrs")).Times(1)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     "asia-east1",
		StorageClass: regionalStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project, "asia-east1-b")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketNilSourceFileBucketMetadataDefaultBucketCreated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "proJect1"
	expectedBucket := "project1-daisy-bkt-us"
	expectedRegion := "US"
	ctx := context.Background()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(nil, nil).Times(1)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	c := ScratchBucketCreator{mockStorageClient, ctx, createMockBucketIteratorWithRandomBuckets(mockCtrl, &ctx, mockStorageClient, project)}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project, "")
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketErrorWhenIteratingOverProjectBucketsWhileCreatingBucketBasedOnSourceFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"

	sourceBucketAttrs := &storage.BucketAttrs{
		Name:         "sourcebucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(sourceBucketAttrs, nil).Times(1)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	mockBucketIterator.EXPECT().Next().Return(nil, fmt.Errorf("iterator error"))

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	_, _, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectID, "")
	assert.NotNil(t, err)
	assert.Equal(t, "iterator error", err.Error())
}

func TestCreateScratchBucketReturnsExistingScratchBucketNoCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"

	sourceBucketAttrs := &storage.BucketAttrs{
		Name:         "sourcebucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	anotherBucketAttrs := &storage.BucketAttrs{
		Name:         "anotherbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}

	scratchBucketAttrs := &storage.BucketAttrs{
		Name:         "project1-daisy-bkt-us-west2",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(sourceBucketAttrs, nil).Times(1)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(sourceBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(scratchBucketAttrs, nil)

	gomock.InOrder(first, second, third)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectID, "")
	assert.Equal(t, "project1-daisy-bkt-us-west2", bucket)
	assert.Equal(t, "us-west2", region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketErrorWhenCreatingBucket(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	project := "PROJECT1"

	sourceBucketAttrs := &storage.BucketAttrs{
		Name:         "sourcebucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	anotherBucketAttrs := &storage.BucketAttrs{
		Name:         "anotherbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}

	scratchBucketAttrs := &storage.BucketAttrs{
		Name:         "project1-daisy-bkt-us-west2",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetBucketAttrs("sourcebucket").
		Return(sourceBucketAttrs, nil).
		Times(1)
	mockStorageClient.EXPECT().
		CreateBucket("project1-daisy-bkt-us-west2", project, scratchBucketAttrs).
		Return(fmt.Errorf("error creating a bucket")).
		Times(1)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(sourceBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	gomock.InOrder(first, second, third)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, project).
		Return(mockBucketIterator)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	_, _, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project, "")
	assert.NotNil(t, err)
	assert.Equal(t, "error creating a bucket", err.Error())
}

func TestIsBucketInProjectTrue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"

	anotherBucketAttrs := &storage.BucketAttrs{
		Name:         "anotherbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}
	scratchBucketAttrs := &storage.BucketAttrs{
		Name:         "project1-daisy-bkt-us-west2",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(scratchBucketAttrs, nil)

	gomock.InOrder(first, second)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	result := c.IsBucketInProject(projectID, "project1-daisy-bkt-us-west2")
	assert.True(t, result)
}

func TestIsBucketInProjectFalseOnNoBucket(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"

	anotherBucketAttrs := &storage.BucketAttrs{
		Name:         "anotherbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}
	scratchBucketAttrs := &storage.BucketAttrs{
		Name:         "some-other-bucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(scratchBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	result := c.IsBucketInProject(projectID, "project1-daisy-bkt-us-west2")
	assert.False(t, result)
}

func TestIsBucketInProjectFalseOnIteratorError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(nil, fmt.Errorf(""))
	gomock.InOrder(first)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	result := c.IsBucketInProject(projectID, "project1-daisy-bkt-us-west2")
	assert.False(t, result)
}

func createMockBucketIteratorWithRandomBuckets(mockCtrl *gomock.Controller, ctx *context.Context,
	storageClient domain.StorageClientInterface,
	project string) domain.BucketIteratorCreatorInterface {
	firstBucketAttrs := &storage.BucketAttrs{
		Name:         "firstbucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	secondBucketAttrs := &storage.BucketAttrs{
		Name:         "secondbucket",
		Location:     "EU",
		StorageClass: "multi_regional",
	}
	thirdBucketAttrs := &storage.BucketAttrs{
		Name:         "thirdbucket",
		Location:     "europe-north1",
		StorageClass: "regional",
	}

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(firstBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(secondBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(thirdBucketAttrs, nil)
	fourth := mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third, fourth)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(*ctx, storageClient, project).
		Return(mockBucketIterator).
		Times(1)
	return mockBucketIteratorCreator
}

func assertErrorFromCreateScratchBucket(t *testing.T, bucket string, region string, err error) {
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", region)
	assert.NotNil(t, err)
}
