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

package storageutils

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
	"io"
	"net/http"
	"testing"
)

func TestGetBucketAttrs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bucket := "a_bucket"
	expected := &storage.BucketAttrs{Name: bucket, Location: "a_region"}
	bucketAttrsStr := "{\"Name\": \"a_bucket\", \"Location\": \"a_region\"}"
	bucketAttrsBytes := []byte(bucketAttrsStr)

	mockResponseBody := mocks.NewMockReadCloser(mockCtrl)
	mockResponseBody.EXPECT().Close()
	mockResponseBody.EXPECT().
		Read(gomock.Any()).
		Return(len(bucketAttrsBytes), io.EOF).
		Do(func(p []byte) {
			copy(p, bucketAttrsBytes)
		})

	mockHTTPClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHTTPClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{StorageClient: nil, HTTPClient: mockHTTPClient, Ctx: context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expected, result)
}

func TestGetBucketAttrsReturnsErrorWhenHttpGetReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bucket := "a_bucket"

	mockResponseBody := mocks.NewMockReadCloser(mockCtrl)
	mockResponseBody.EXPECT().Close()

	mockHTTPClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHTTPClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK},
			fmt.Errorf("some error"))

	sc := StorageClient{StorageClient: nil, HTTPClient: mockHTTPClient, Ctx: context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestGetBucketAttrsReturnsErrorWhenHttpStatusNotOK(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bucket := "a_bucket"

	mockResponseBody := mocks.NewMockReadCloser(mockCtrl)
	mockResponseBody.EXPECT().Close()

	mockHTTPClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHTTPClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusBadRequest}, nil)

	sc := StorageClient{StorageClient: nil, HTTPClient: mockHTTPClient, Ctx: context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestGetBucketAttrsReturnsErrorWhenReadingFails(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bucket := "a_bucket"

	mockResponseBody := mocks.NewMockReadCloser(mockCtrl)
	mockResponseBody.EXPECT().Close()
	mockResponseBody.EXPECT().
		Read(gomock.Any()).
		Return(0, fmt.Errorf("some error"))

	mockHTTPClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHTTPClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{StorageClient: nil, HTTPClient: mockHTTPClient, Ctx: context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestGetBucketAttrsReturnsErrorInvalidJson(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bucket := "a_bucket"
	bucketAttrsStr := "NOT__VALID_JSON"
	bucketAttrsBytes := []byte(bucketAttrsStr)

	mockResponseBody := mocks.NewMockReadCloser(mockCtrl)
	mockResponseBody.EXPECT().Close()
	mockResponseBody.EXPECT().
		Read(gomock.Any()).
		Return(len(bucketAttrsBytes), io.EOF).
		Do(func(p []byte) {
			copy(p, bucketAttrsBytes)
		})

	mockHTTPClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHTTPClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{StorageClient: nil, HTTPClient: mockHTTPClient, Ctx: context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}

func TestDeleteGcsPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	mockStorageObjectDeleter := mocks.NewMockStorageObjectDeleterInterface(mockCtrl)
	firstDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile1.txt")
	secondDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile2.txt")
	gomock.InOrder(firstDeletion, secondDeletion)

	sc := StorageClient{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.Nil(t, err)
}

func TestDeleteGcsPathErrorWhenInvalidGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sc := StorageClient{}
	err := sc.DeleteGcsPath("NOT_GCS_PATH")
	assert.NotNil(t, err)
}

func TestDeleteGcsPathErrorWhenIteratorReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	mockObjectIterator.EXPECT().Next().Return(nil, fmt.Errorf("iterator error"))

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
}

func TestDeleteGcsPathErrorWhenErrorDeletingAFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	gomock.InOrder(first, second)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	mockStorageObjectDeleter := mocks.NewMockStorageObjectDeleterInterface(mockCtrl)
	firstDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile1.txt").Return(nil)
	secondDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile2.txt").Return(fmt.Errorf("can't delete second file"))
	gomock.InOrder(firstDeletion, secondDeletion)

	sc := StorageClient{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
}


func TestFindGcsFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/bingo.ovf"}, nil)
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator}
	objectHandle, err := sc.FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", objectHandle.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileNoFileFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile4"}, nil)
	fourth := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile5.txt"}, nil)
	fifth := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third, fourth, fifth)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator}
	objectHandle, err := sc.FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, objectHandle)
	assert.NotNil(t, err)
}

func TestFindGcsFileInvalidGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sc := StorageClient{}
	objectHandle, err := sc.FindGcsFile("NOT_A_GCS_PATH", ".ovf")
	assert.Nil(t, objectHandle)
	assert.NotNil(t, err)
}

func TestFindGcsFileErrorWhileIteratingThroughFilesInPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().Return(nil, fmt.Errorf("error while iterating"))
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator}
	objectHandle, err := sc.FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, objectHandle)
	assert.NotNil(t, err)
}