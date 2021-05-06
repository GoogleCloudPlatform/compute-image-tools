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

package importer

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestEmptyFilesAreRejected(t *testing.T) {
	source := fileSource{
		gcsPath: "gs://bucket/global/images/ubuntu-1604",
		bucket:  "bucket",
		object:  "global/images/ubuntu-1604",
	}

	fileContent := ""

	factory := NewSourceFactory(createMockStorageClient(t, source, fileContent, true))
	_, err := factory.Init(source.Path(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot import an image from an empty file")
}

func TestGzipCompressedFilesAreRejected(t *testing.T) {
	source := fileSource{
		gcsPath: "gs://bucket/global/images/ubuntu-1604",
		bucket:  "bucket",
		object:  "global/images/ubuntu-1604",
	}

	fileContent := test.CreateCompressedFile()

	factory := NewSourceFactory(createMockStorageClient(t, source, fileContent, true))
	_, err := factory.Init(source.Path(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the input file is a gzip file, which is not supported")
}

func TestUncompressedFilesAreAllowed(t *testing.T) {
	source := fileSource{
		gcsPath: "gs://bucket/global/images/ubuntu-1604",
		bucket:  "bucket",
		object:  "global/images/ubuntu-1604",
	}

	fileContent := "fileContent"

	factory := NewSourceFactory(createMockStorageClient(t, source, fileContent, true))
	result, err := factory.Init(source.Path(), "")
	assert.NoError(t, err)
	assert.Equal(t, result, source)
}

func TestGcsFilePathMustBeFullyQualified(t *testing.T) {
	for _, invalidPath := range []string{"file.vmdk", "gs://bucket", "gs://bucket/"} {
		_, err := NewSourceFactory(nil).Init(invalidPath, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not a valid Cloud Storage object path")
	}
}

func TestEnsureEitherFileOrImageIsPresent(t *testing.T) {
	var cases = []struct {
		name  string
		file  fileSource
		image string
		valid bool
		verifyFileRead bool
	}{
		{
			name: "both",
			file: fileSource{
				gcsPath: "gs://bucket/global/images/ubuntu-1604",
				bucket:  "bucket",
				object:  "global/images/ubuntu-1604",
			},
			image: "global/images/ubuntu-1604",
			valid: false,
			verifyFileRead: false,
		},
		{
			name:  "neither",
			valid: false,
			verifyFileRead: false,
		},
		{
			name: "only file",
			file: fileSource{
				gcsPath: "gs://bucket/global/images/ubuntu-1604",
				bucket:  "bucket",
				object:  "global/images/ubuntu-1604",
			},
			valid: true,
			verifyFileRead: true,
		},
		{
			name:  "only image",
			image: "global/images/ubuntu-1604",
			valid: true,
			verifyFileRead: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewSourceFactory(createMockStorageClient(t, tt.file, "file-content", tt.verifyFileRead))
			_, err := factory.Init(tt.file.Path(), tt.image)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "either -source_file or -source_image has to be specified")
			}
		})
	}
}

func TestUnqualifiedImagePathsAreGlobalized(t *testing.T) {
	var cases = []struct {
		originalURI string
		expectedURI string
	}{
		{"ubuntu-1604", "global/images/ubuntu-1604"},
		{"projects/daisy/global/images/ubuntu-1604", "projects/daisy/global/images/ubuntu-1604"},
	}

	for _, tt := range cases {
		t.Run(tt.originalURI, func(t *testing.T) {
			source, err := NewSourceFactory(nil).Init("", tt.originalURI)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURI, source.Path())
		})

	}
}

func TestImagePathIsValidated(t *testing.T) {
	var cases = []struct {
		originalURI string
		errMessage  string
	}{
		{"file.vmdk", "invalid image name"},
		{"gs://bucket/file.vmdk", "invalid image reference"},
		{strings.Repeat("a", 80), "Image name must be 1-63 characters long"},
		{"global/images/ubuntu/", "Image name must be 1-63 characters long"},
	}

	for _, tt := range cases {

		t.Run(tt.originalURI, func(t *testing.T) {
			_, err := NewSourceFactory(nil).Init("", tt.originalURI)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMessage)
		})

	}
}

func createMockStorageClient(t *testing.T, filePath fileSource, fileContent string, testExpectations bool) *mocks.MockStorageClientInterface {
	mockCtrl := gomock.NewController(t)
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	if testExpectations {
		mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(strings.NewReader(fileContent)), nil)
		mockStorageClient.EXPECT().GetObject(filePath.bucket, filePath.object).Return(mockStorageObject)
	}
	return mockStorageClient
}
