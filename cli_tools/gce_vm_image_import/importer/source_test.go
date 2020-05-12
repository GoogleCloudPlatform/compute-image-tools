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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/test"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestEmptyFilesAreRejected(t *testing.T) {
	source := fileSource{
		gcsPath: "gs://bucket/global/images/ubuntu-1604",
		bucket:  "bucket",
		object:  "global/images/ubuntu-1604",
	}

	fileContent := ""

	_, err := initAndValidateSource(source.path(), "",
		createMockStorageClient(t, source, fileContent))
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

	_, err := initAndValidateSource(source.path(), "",
		createMockStorageClient(t, source, fileContent))
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

	result, err := initAndValidateSource(source.path(), "",
		createMockStorageClient(t, source, fileContent))
	assert.NoError(t, err)
	assert.Equal(t, result, source)
}

func TestGcsFilePathMustBeFullyQualified(t *testing.T) {
	for _, invalidPath := range []string{"file.vmdk", "gs://bucket", "gs://bucket/"} {
		_, err := initAndValidateSource(invalidPath, "", nil)
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
		},
		{
			name:  "neither",
			valid: false,
		},
		{
			name: "only file",
			file: fileSource{
				gcsPath: "gs://bucket/global/images/ubuntu-1604",
				bucket:  "bucket",
				object:  "global/images/ubuntu-1604",
			},
			valid: true,
		},
		{
			name:  "only image",
			image: "global/images/ubuntu-1604",
			valid: true,
		},
	}

	for _, tt := range cases {

		t.Run(tt.name, func(t *testing.T) {
			_, err := initAndValidateSource(tt.file.path(), tt.image,
				createMockStorageClient(t, tt.file, "file-content"))
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
		input          string
		expectedResult string
	}{
		{"ubuntu-1604", "global/images/ubuntu-1604"},
		{"projects/daisy/global/images/ubuntu-1604", "projects/daisy/global/images/ubuntu-1604"},
	}

	for _, tt := range cases {

		t.Run(tt.input, func(t *testing.T) {
			source, err := initAndValidateSource("", tt.input, nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, source.path())
		})

	}
}

func TestImagePathIsValidated(t *testing.T) {
	var cases = []struct {
		path       string
		errMessage string
	}{
		{"file.vmdk", "invalid image name"},
		{"gs://bucket/file.vmdk", "invalid image reference"},
		{strings.Repeat("a", 80), "Image name must be 1-63 characters long"},
		{"global/images/ubuntu/", "Image name must be 1-63 characters long"},
	}

	for _, tt := range cases {

		t.Run(tt.path, func(t *testing.T) {
			_, err := initAndValidateSource("", tt.path, nil)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMessage)
		})

	}
}

func createMockStorageClient(t *testing.T, filePath fileSource, fileContent string) *mocks.MockStorageClientInterface {
	mockCtrl := gomock.NewController(t)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().GetObjectReader(filePath.bucket, filePath.object).Return(
		ioutil.NopCloser(strings.NewReader(fileContent)), nil)
	return mockStorageClient
}
