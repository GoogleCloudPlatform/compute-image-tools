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

package gcevmimageimportutil

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
	"testing"
)

func TestCreateScratchBucketErrorWhenProjectNotProvided(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("", "")
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func TestCreateScratchBucketNoSourceFileDefaultBucketCreated(t *testing.T) {
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
	bucket, region, err := c.CreateScratchBucket("", project)
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
	bucket, region, err := c.CreateScratchBucket("", project)
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", region)
	assert.NotNil(t, err)
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
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project)
	assert.Equal(t, "project1-daisy-bkt-us-west2", bucket)
	assert.Equal(t, "us-west2", region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketInvalidSourceFileDefaultBucketCreated(t *testing.T) {
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
	bucket, region, err := c.CreateScratchBucket("NOT_A_GS_PATH", project)
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
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
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project)
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
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project)
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketErrorWhenIteratingOverProjectBucketsWhileCreatingBucketBasedOnSourceFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectID := "PROJECT1"
	expectedBucket := "project1-daisy-bkt-us"
	expectedRegion := "US"

	sourceBucketAttrs := &storage.BucketAttrs{
		Name:         "sourcebucket",
		Location:     "us-west2",
		StorageClass: "regional",
	}
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(sourceBucketAttrs, nil).Times(1)
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, projectID, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	mockBucketIterator.EXPECT().Next().Return(nil, fmt.Errorf("iterator error"))
	secondMockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	secondMockBucketIterator.EXPECT().Next().Return(sourceBucketAttrs, nil)
	secondMockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	first := mockBucketIteratorCreator.EXPECT().CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(mockBucketIterator)
	second := mockBucketIteratorCreator.EXPECT().CreateBucketIterator(ctx, mockStorageClient, projectID).
		Return(secondMockBucketIterator)

	gomock.InOrder(first, second)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectID)
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
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
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectID)
	assert.Equal(t, "project1-daisy-bkt-us-west2", bucket)
	assert.Equal(t, "us-west2", region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketErrorWhenCreatingBucket(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	project := "PROJECT1"
	expectedBucket := "project1-daisy-bkt-us"
	expectedRegion := "US"

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
	mockStorageClient.EXPECT().CreateBucket(expectedBucket, project, &storage.BucketAttrs{
		Name:         expectedBucket,
		Location:     defaultRegion,
		StorageClass: defaultStorageClass,
	}).Return(nil)

	mockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first := mockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second := mockBucketIterator.EXPECT().Next().Return(sourceBucketAttrs, nil)
	third := mockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	gomock.InOrder(first, second, third)

	secondMockBucketIterator := mocks.NewMockBucketIteratorInterface(mockCtrl)
	first = secondMockBucketIterator.EXPECT().Next().Return(anotherBucketAttrs, nil)
	second = secondMockBucketIterator.EXPECT().Next().Return(nil, iterator.Done)

	gomock.InOrder(first, second)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, project).
		Return(mockBucketIterator)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, project).
		Return(secondMockBucketIterator)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", project)
	assert.Equal(t, expectedBucket, bucket)
	assert.Equal(t, expectedRegion, region)
	assert.Nil(t, err)
}

func createMockBucketIteratorWithRandomBuckets(mockCtrl *gomock.Controller, ctx *context.Context,
	storageClient commondomain.StorageClientInterface,
	project string) commondomain.BucketIteratorCreatorInterface {
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
