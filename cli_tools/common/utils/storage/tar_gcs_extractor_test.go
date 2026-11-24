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
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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
	mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(testTarFileReader), nil)

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
	mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(testTarFileReader), nil)

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
	mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(testTarFileReader), nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.tar").
		Return(mockStorageObject)

	tge := TarGcsExtractor{ctx: ctx, storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}
	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.tar", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}

// buildTar returns an in-memory tar archive containing one regular file per name.
func buildTar(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, name := range names {
		content := []byte("content of " + name)
		if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(content))}); err != nil {
			t.Fatalf("WriteHeader(%q) failed: %v", name, err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("Write(%q) failed: %v", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close() failed: %v", err)
	}
	return buf.Bytes()
}

func newExtractorForTar(mockCtrl *gomock.Controller, tarBytes []byte) (*TarGcsExtractor, *mocks.MockStorageClientInterface) {
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().NewReader().Return(ioutil.NopCloser(bytes.NewReader(tarBytes)), nil)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().
		GetObject("sourcebucket", "sourcepath/sometar.ova").
		Return(mockStorageObject)

	return &TarGcsExtractor{ctx: context.Background(), storageClient: mockStorageClient, logger: logging.NewToolLogger("[import-ovf]")}, mockStorageClient
}

func TestExtractTarToGcsAcceptsFlatAndDotSlashNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tge, mockStorageClient := newExtractorForTar(mockCtrl, buildTar(t, "vm.ovf", "./disk1.vmdk", "disk..2.vmdk"))
	gomock.InOrder(
		mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/vm.ovf", gomock.Any()).Return(nil),
		mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/disk1.vmdk", gomock.Any()).Return(nil),
		mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/disk..2.vmdk", gomock.Any()).Return(nil),
	)

	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.ova", "gs://destbucket/destpath/")

	assert.Nil(t, err)
}

func TestExtractTarToGcsRejectsUnsafeMemberNames(t *testing.T) {
	for _, name := range []string{
		"../evil.sh",
		"../../other-build/daisy-files/startup.sh",
		"./../evil.sh",
		"..",
		".",
		"/abs/evil.sh",
		"sub/disk.vmdk",
		`..\evil.sh`,
		`sub\disk.vmdk`,
	} {
		t.Run(name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			// No WriteToGCS expectation: any write fails the test.
			tge, _ := newExtractorForTar(mockCtrl, buildTar(t, name))

			err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.ova", "gs://destbucket/destpath/")

			assert.NotNil(t, err)
		})
	}
}

func TestExtractTarToGcsStopsBeforeWritingUnsafeMember(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tge, mockStorageClient := newExtractorForTar(mockCtrl, buildTar(t, "vm.ovf", "../evil.sh", "disk1.vmdk"))
	mockStorageClient.EXPECT().WriteToGCS("destbucket", "destpath/vm.ovf", gomock.Any()).Return(nil)

	err := tge.ExtractTarToGcs("gs://sourcebucket/sourcepath/sometar.ova", "gs://destbucket/destpath/")

	assert.NotNil(t, err)
}

func TestSafeDestinationFilePath(t *testing.T) {
	for _, tc := range []struct {
		destinationPath string
		memberName      string
		want            string
	}{
		{"destpath/", "vm.ovf", "destpath/vm.ovf"},
		{"destpath", "./vm.ovf", "destpath/vm.ovf"},
		{"a/b/ovf/", "disk 1.vmdk", "a/b/ovf/disk 1.vmdk"},
		{"", "vm.ovf", "vm.ovf"},
	} {
		got, err := safeDestinationFilePath(tc.destinationPath, tc.memberName)
		assert.Nil(t, err, fmt.Sprintf("safeDestinationFilePath(%q, %q)", tc.destinationPath, tc.memberName))
		assert.Equal(t, tc.want, got)
	}
}
