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

package ovfexporter

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestManifestGenerator_Generate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := prepareMocksForSuccessfulGenerate(mockCtrl)
	g := createManifestFileGenerator(mockStorageClient)
	manifestContent, err := g.generate("a-bucket", "folder1/subfolder/")
	assert.Nil(t, err)
	assert.Equal(t, manifestContent, "SHA1(afile1.txt)= ef664589aac768e7deb904c687f8ff9ca6cc2100\nSHA1(afile2.txt)= f7343ea75dd828dcbe1dce4747e980cdbb9758c1\n")
}

func TestManifestGenerator_GenerateAndWriteToGCS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := prepareMocksForSuccessfulGenerate(mockCtrl)
	mockStorageClient.EXPECT().WriteToGCS(
		"a-bucket", "folder1/subfolder/a-manifest.mf",
		strings.NewReader("SHA1(afile1.txt)= ef664589aac768e7deb904c687f8ff9ca6cc2100\nSHA1(afile2.txt)= f7343ea75dd828dcbe1dce4747e980cdbb9758c1\n"))

	g := createManifestFileGenerator(mockStorageClient)
	err := g.GenerateAndWriteToGCS("gs://a-bucket/folder1/subfolder/", "a-manifest.mf")
	assert.Nil(t, err)
}

func prepareMocksForSuccessfulGenerate(mockCtrl *gomock.Controller) *mocks.MockStorageClientInterface {
	aFile1StorageObject := mocks.NewMockStorageObject(mockCtrl)
	aFile1StorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(strings.NewReader("aFile1Content")), nil)
	aFile2StorageObject := mocks.NewMockStorageObject(mockCtrl)
	aFile2StorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(strings.NewReader("aFile2Content")), nil)

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "folder1/subfolder/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "folder1/subfolder/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjects("a-bucket", "folder1/subfolder/").
		Return(mockObjectIterator)
	mockStorageClient.EXPECT().
		GetObject("a-bucket", "folder1/subfolder/afile1.txt").
		Return(aFile1StorageObject)
	mockStorageClient.EXPECT().
		GetObject("a-bucket", "folder1/subfolder/afile2.txt").
		Return(aFile2StorageObject)
	return mockStorageClient
}

func TestManifestGenerator_GenerateAndWriteToGCS_ErrorOnInvalidGCSPath(t *testing.T) {
	g := createManifestFileGenerator(nil)
	err := g.GenerateAndWriteToGCS("NOT_A_GCS_PATH", "a-manifest.mf")
	assert.Equal(t, daisy.Errf("%q is not a valid Cloud Storage path", "NOT_A_GCS_PATH"), err)
}

func TestManifestGenerator_Generate_ErrorOnGCSIteration(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	iteratorError := fmt.Errorf("iterator error")
	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	mockObjectIterator.EXPECT().Next().Return(nil, iteratorError)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjects("a-bucket", "folder1/subfolder/").
		Return(mockObjectIterator)

	g := createManifestFileGenerator(mockStorageClient)
	manifestContent, err := g.generate("a-bucket", "folder1/subfolder/")
	assert.Empty(t, manifestContent)
	assert.Equal(t, iteratorError, err)
}

func TestManifestGenerator_Generate_ErrorOnFileReading(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fileReadingError := fmt.Errorf("file reading error")
	aFile1StorageObject := mocks.NewMockStorageObject(mockCtrl)
	aFile1StorageObject.EXPECT().NewReader().Return(nil, fileReadingError)

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().Return(&storage.ObjectAttrs{Name: "folder1/subfolder/afile1.txt"}, nil)
	gomock.InOrder(first)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjects("a-bucket", "folder1/subfolder/").
		Return(mockObjectIterator)
	mockStorageClient.EXPECT().
		GetObject("a-bucket", "folder1/subfolder/afile1.txt").
		Return(aFile1StorageObject)

	g := createManifestFileGenerator(mockStorageClient)
	manifestContent, err := g.generate("a-bucket", "folder1/subfolder/")
	assert.Empty(t, manifestContent)
	assert.Equal(t, fileReadingError, err)
}

func TestManifestGenerator_GenerateAndWriteToGCS_ErrorOnGenerate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	iteratorError := fmt.Errorf("iterator error")
	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	mockObjectIterator.EXPECT().Next().Return(nil, iteratorError)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObjects("a-bucket", "folder1/subfolder/").
		Return(mockObjectIterator)

	g := createManifestFileGenerator(mockStorageClient)
	err := g.GenerateAndWriteToGCS("gs://a-bucket/folder1/subfolder/", "a-manifest.mf")
	assert.Equal(t, iteratorError, err)
}

func TestManifestGenerator_GenerateAndWriteToGCS_ErrorOnWriteToGCS(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	gcsWriteError := fmt.Errorf("gcs write error")

	mockStorageClient := prepareMocksForSuccessfulGenerate(mockCtrl)
	mockStorageClient.EXPECT().WriteToGCS(
		"a-bucket", "folder1/subfolder/a-manifest.mf",
		strings.NewReader("SHA1(afile1.txt)= ef664589aac768e7deb904c687f8ff9ca6cc2100\nSHA1(afile2.txt)= f7343ea75dd828dcbe1dce4747e980cdbb9758c1\n")).Return(gcsWriteError)

	g := createManifestFileGenerator(mockStorageClient)
	err := g.GenerateAndWriteToGCS("gs://a-bucket/folder1/subfolder/", "a-manifest.mf")
	assert.Equal(t, gcsWriteError, err)
}

func createManifestFileGenerator(storageClient domain.StorageClientInterface) *ovfManifestGeneratorImpl {
	return &ovfManifestGeneratorImpl{
		storageClient: storageClient,
		cancelChan:    make(chan string),
	}
}
