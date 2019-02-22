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
	"bufio"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestExtractTarToGcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	testTarFile, _ := os.Open("../../../test_data/test_tar.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(ioutil.NopCloser(testTarFileReader), nil)
	first := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file1.txt", gomock.Any()).Return(nil)
	second := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file2.txt", gomock.Any()).Return(nil)

	gomock.InOrder(first, second)

	tge := TarGcsExtractor{ctx, mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.Nil(t, err)
}

func TestExtractTarToGcsErrorWhenInvalidSourceGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	tge := TarGcsExtractor{context.Background(), mockStorageClient}
	err := tge.ExtractTarToGcs("NOT_GCS_PATH", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenNonExistentSourceGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(nil, fmt.Errorf("no file"))

	tge := TarGcsExtractor{context.Background(), mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenInvalidDestinationPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockReader := mocks.NewMockReadCloser(mockCtrl)
	mockReader.EXPECT().Close()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(mockReader, nil)

	tge := TarGcsExtractor{context.Background(), mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "NOT_GCS_PATH")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenWriteToGCSFailed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	testTarFile, _ := os.Open("../../../test_data/test_tar.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(ioutil.NopCloser(testTarFileReader), nil)
	first := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file1.txt", gomock.Any()).Return(nil)
	second := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file2.txt", gomock.Any()).Return(fmt.Errorf("error writing to gcs"))

	gomock.InOrder(first, second)

	tge := TarGcsExtractor{ctx, mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenErrorReadingTarFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockReader := mocks.NewMockReadCloser(mockCtrl)
	mockReader.EXPECT().Read(gomock.Any()).Return(0, fmt.Errorf("error reading tar"))
	mockReader.EXPECT().Close()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(mockReader, nil)

	tge := TarGcsExtractor{context.Background(), mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorIfDirPresentInTar(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	testTarFile, _ := os.Open("../../../test_data/test_tar_with_dir.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjectReader("sourcebucket", "sourcepath/sometar.tar").
		Return(ioutil.NopCloser(testTarFileReader), nil)

	tge := TarGcsExtractor{ctx, mockStorageClient}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}
