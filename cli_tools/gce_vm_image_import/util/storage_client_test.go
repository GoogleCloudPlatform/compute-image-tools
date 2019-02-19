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

	mockHttpClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHttpClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{nil, mockHttpClient, context.Background()}
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

	mockHttpClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHttpClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK},
			fmt.Errorf("some error"))

	sc := StorageClient{nil, mockHttpClient, context.Background()}
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

	mockHttpClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHttpClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusBadRequest}, nil)

	sc := StorageClient{nil, mockHttpClient, context.Background()}
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

	mockHttpClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHttpClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{nil, mockHttpClient, context.Background()}
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

	mockHttpClient := mocks.NewMockHttpClientInterface(mockCtrl)
	mockHttpClient.EXPECT().
		Get("https://www.googleapis.com/storage/v1/b/"+bucket+"?fields=location%2CstorageClass").
		Return(&http.Response{Body: mockResponseBody, StatusCode: http.StatusOK}, nil)

	sc := StorageClient{nil, mockHttpClient, context.Background()}
	result, err := sc.GetBucketAttrs(bucket)
	assert.NotNil(t, err)
	assert.Nil(t, result)
}
