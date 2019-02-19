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
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
	"testing"
)

func TestCreateScratchBucketNoSourceFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("", "project1")
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", region)
	assert.Nil(t, err)
}

func TestCreateScratchBucketNewBucketCreatedProjectAsFlag(t *testing.T) {
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
	mockStorageClient.EXPECT().CreateBucket("project1-daisy-bkt-us-west2", ctx, project, scratchBucketAttrs).Return(nil).Times(1)

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

func TestCreateScratchBucketInvalidSourceFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	ctx := context.Background()

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("NOT_A_GS_PATH", "PROJECT1")
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func TestCreateScratchBucketErrorRetrievingSourceFileBucketMetadata(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(&storage.BucketAttrs{}, fmt.Errorf("error retrieving bucket attrs")).Times(1)

	ctx := context.Background()

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", "PROJECT1")
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func TestCreateScratchBucketNilSourceFileBucketMetadata(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetBucketAttrs("sourcebucket").Return(nil, nil).Times(1)

	ctx := context.Background()

	c := ScratchBucketCreator{mockStorageClient, ctx, nil}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", "PROJECT1")
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func assertErrorFromCreateScratchBucket(t *testing.T, bucket string, region string, err error) {
	assert.Equal(t, "", bucket)
	assert.Equal(t, "", region)
	assert.NotNil(t, err)
}

func TestCreateScratchBucketErrorWhenIteratingOverProjectBuckets(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectId := "PROJECT1"

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
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectId).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectId)
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}

func TestCreateScratchBucketReturnsExistingScratchBucketNoCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ctx := context.Background()

	projectId := "PROJECT1"

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

	gomock.InOrder(
		first,
		second,
		third,
	)

	mockBucketIteratorCreator := mocks.NewMockBucketIteratorCreatorInterface(mockCtrl)
	mockBucketIteratorCreator.EXPECT().
		CreateBucketIterator(ctx, mockStorageClient, projectId).
		Return(mockBucketIterator).
		Times(1)

	c := ScratchBucketCreator{mockStorageClient, ctx, mockBucketIteratorCreator}
	bucket, region, err := c.CreateScratchBucket("gs://sourcebucket/sourcefile", projectId)
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
		CreateBucket("project1-daisy-bkt-us-west2", ctx, project, scratchBucketAttrs).
		Return(fmt.Errorf("error creating a bucket")).
		Times(1)

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
	assertErrorFromCreateScratchBucket(t, bucket, region, err)
}
