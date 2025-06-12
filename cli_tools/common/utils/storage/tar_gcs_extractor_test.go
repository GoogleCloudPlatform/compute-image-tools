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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestExtractTarToGcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	testTarFile, _ := os.Open("../../../test_data/test_tar.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(io.NopCloser(testTarFileReader), nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	first := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file1.txt", gomock.Any()).Return(nil)
	second := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file2.txt", gomock.Any()).Return(nil)
	third := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file with spaces.txt", gomock.Any()).Return(nil)

	gomock.InOrder(first, second, third)

	tge := TarGcsExtractor{ctx: ctx, storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.Nil(t, err)
}

func TestExtractTarToGcsErrorWhenInvalidSourceGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	tge := TarGcsExtractor{ctx: context.Background(), storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("NOT_GCS_PATH", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenNonExistentSourceGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(nil, fmt.Errorf("no file"))

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	tge := TarGcsExtractor{ctx: context.Background(), storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenInvalidDestinationPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockReader := mocks.NewMockReadCloser(mockCtrl)
	mockReader.EXPECT().Close()

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(mockReader, nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	tge := TarGcsExtractor{ctx: context.Background(), storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "NOT_GCS_PATH")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenWriteToGCSFailed(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	testTarFile, _ := os.Open("../../../test_data/test_tar.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(io.NopCloser(testTarFileReader), nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	first := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file1.txt", gomock.Any()).Return(nil)
	second := mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/file2.txt", gomock.Any()).Return(fmt.Errorf("error writing to gcs"))

	gomock.InOrder(first, second)

	tge := TarGcsExtractor{ctx: ctx, storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorWhenErrorReadingTarFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockReader := mocks.NewMockReadCloser(mockCtrl)
	mockReader.EXPECT().Read(gomock.Any()).Return(0, fmt.Errorf("error reading tar"))
	mockReader.EXPECT().Close()

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(mockReader, nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	tge := TarGcsExtractor{ctx: context.Background(), storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")
	assert.NotNil(t, err)
}

func TestExtractTarToGcsErrorIfDirPresentInTar(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	testTarFile, _ := os.Open("../../../test_data/test_tar_with_dir.tar")
	testTarFileReader := bufio.NewReader(testTarFile)

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(io.NopCloser(testTarFileReader), nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	tge := TarGcsExtractor{ctx: ctx, storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}
