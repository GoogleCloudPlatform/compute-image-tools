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
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

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

	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	mockStorageObject.EXPECT().Delete().Return(nil).AnyTimes()
	mockStorageObjectCreator := mocks.NewMockStorageObjectCreatorInterface(mockCtrl)
	mockStorageObjectCreator.EXPECT().
		GetObject("sourcebucket", "sourcepath/furtherpath/afile1.txt").
		Return(mockStorageObject)
	mockStorageObjectCreator.EXPECT().
		GetObject("sourcebucket", "sourcepath/furtherpath/afile2.txt").
		Return(mockStorageObject)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Soc: mockStorageObjectCreator,
		Logger: logging.NewToolLogger("[test]")}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.Nil(t, err)
}

func TestDeleteGcsPathErrorWhenInvalidGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sc := Client{}
	err := sc.DeleteGcsPath("NOT_GCS_PATH")
	assert.NotNil(t, err)
}

func TestDeleteGcsPathErrorWhenIteratorReturnsError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	mockObjectIterator.EXPECT().Next().Return(nil, fmt.Errorf("iterator error"))

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator(
		"sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
}

func TestDeleteGcsPathErrorWhenErrorDeletingAFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	gomock.InOrder(first, second)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	mockStorageObjectCreator := mocks.NewMockStorageObjectCreatorInterface(mockCtrl)
	mockStorageObject := mocks.NewMockStorageObject(mockCtrl)
	firstObject := mockStorageObject.EXPECT().Delete().Return(nil)
	secondObject := mockStorageObject.EXPECT().Delete().Return(fmt.Errorf("can't delete second file"))
	mockStorageObjectCreator.EXPECT().
		GetObject("sourcebucket", "sourcepath/furtherpath/afile1.txt").Return(mockStorageObject)
	mockStorageObjectCreator.EXPECT().
		GetObject("sourcebucket", "sourcepath/furtherpath/afile2.txt").Return(mockStorageObject)
	gomock.InOrder(firstObject, secondObject)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Soc: mockStorageObjectCreator,
		Logger: logging.NewToolLogger("[test]")}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
}

func TestDeleteObject(t *testing.T) {

	tests := []struct {
		name           string
		objectURI      string
		bucket         string
		path           string
		deleteExpected bool
		deleteError    error
		errorExpected  string
	}{
		{
			name:           "return nil when successful deletion",
			objectURI:      "gs://bucket/path",
			bucket:         "bucket",
			path:           "path",
			deleteExpected: true,
		}, {
			name:          "return error when malformed objectURI",
			objectURI:     "bucket//path",
			errorExpected: "Error deleting `bucket//path`: `\"bucket//path\" is not a valid Cloud Storage path`",
		}, {
			name:           "return error when delete RPC fails",
			objectURI:      "gs://bucket/path",
			bucket:         "bucket",
			path:           "path",
			deleteExpected: true,
			deleteError:    errors.New("HTTP 404"),
			errorExpected:  "Error deleting `gs://bucket/path`: `HTTP 404`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorageObjectCreator := mocks.NewMockStorageObjectCreatorInterface(ctrl)

			if tt.deleteExpected {
				mockStorageObject := mocks.NewMockStorageObject(ctrl)
				mockStorageObjectCreator.EXPECT().
					GetObject(tt.bucket, tt.path).Return(mockStorageObject)
				mockStorageObject.EXPECT().Delete().Return(tt.deleteError)
			}

			client := Client{
				Oic:    nil,
				Soc:    mockStorageObjectCreator,
				Logger: logging.NewToolLogger("[test]"),
			}
			err := client.DeleteObject(tt.objectURI)
			if tt.errorExpected == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorExpected)
			}
		})
	}
}

func TestFindGcsFileNoTrailingSlash(t *testing.T) {
	doTestFindGcsFile(t, "sourcebucket", "sourcepath/furtherpath")
}

func TestFindGcsFileTrailingSlash(t *testing.T) {
	doTestFindGcsFile(t, "sourcebucket", "sourcepath/furtherpath/")
}

func doTestFindGcsFile(t *testing.T, bucket, lookupPath string) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	sourcePath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/"}, nil)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/bingo.ovf"}, nil)
	gomock.InOrder(sourcePath, furtherPath, first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator(bucket, lookupPath).
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFile(
		fmt.Sprintf("gs://%v/%v", bucket, lookupPath), ".ovf")

	assert.NotNil(t, storageObject)
	assert.Equal(t, "sourcebucket", storageObject.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", storageObject.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileNoFileFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	sourcePath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/"}, nil)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile4"}, nil)
	fourth := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile5.txt"}, nil)
	fifth := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(sourcePath, furtherPath, first, second, third, fourth, fifth)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, storageObject)
	assert.NotNil(t, err)
}

func TestFindGcsFileInvalidGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sc := Client{}
	storageObject, err := sc.FindGcsFile("NOT_A_GCS_PATH", ".ovf")
	assert.Nil(t, storageObject)
	assert.NotNil(t, err)
}

func TestFindGcsFileErrorWhileIteratingThroughFilesInPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	sourcePath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/"}, nil)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(nil, fmt.Errorf("error while iterating"))
	gomock.InOrder(sourcePath, furtherPath, first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.
		FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, storageObject)
	assert.NotNil(t, err)
}

func TestFindGcsFileDepthLimitedFileInRoot(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "bingo.ovf"}, nil)
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 0)

	assert.NotNil(t, storageObject)
	assert.Equal(t, "sourcebucket", storageObject.BucketName())
	assert.Equal(t, "bingo.ovf", storageObject.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileDepthLimitedFileNotFoundInRoot(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "subfolder/bingo.ovf"}, nil)
	fourth := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "afile4.txt"}, nil)
	done := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third, fourth, done)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 0)

	assert.Nil(t, storageObject)
	assert.NotNil(t, err)
}

func TestFindGcsFileDepthLimitedFileInSubFolderlookupFromRootTrailingSlash(t *testing.T) {
	doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromRoot(t, "gs://sourcebucket/")
}

func TestFindGcsFileDepthLimitedFileInSubFolderlookupFromRootNoTrailingSlash(t *testing.T) {
	doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromRoot(t, "gs://sourcebucket")
}

func doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromRoot(t *testing.T, gcsDirectoryPath string) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	sourcePath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/"}, nil)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/bingo.ovf"}, nil)
	gomock.InOrder(sourcePath, furtherPath, first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFileDepthLimited(
		gcsDirectoryPath, ".ovf", 2)

	assert.NotNil(t, storageObject)
	assert.Equal(t, "sourcebucket", storageObject.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", storageObject.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileDepthLimitedFileInSubFolderlookupFromSubfolderTrailingSlash(t *testing.T) {
	doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromSubfolder(t, "sourcebucket", "sourcepath/furtherpath/")
}

func TestFindGcsFileDepthLimitedFileInSubFolderlookupFromSubfolderNoTrailingSlash(t *testing.T) {
	doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromSubfolder(t, "sourcebucket", "sourcepath/furtherpath")
}

func doTestFindGcsFileDepthLimitedFileInSubFolderlookupFromSubfolder(t *testing.T, bucket, lookupPath string) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/bingo.ovf"}, nil)
	gomock.InOrder(furtherPath, first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator(bucket, lookupPath).
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFileDepthLimited(
		fmt.Sprintf("gs://%v/%v", bucket, lookupPath), ".ovf", 0)

	assert.NotNil(t, storageObject)
	assert.Equal(t, "sourcebucket", storageObject.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", storageObject.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileDepthLimitedFileNotFoundInSubFolder(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	sourcePath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/"}, nil)
	furtherPath := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/"}, nil)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/evenfurtherpath/bingo.ovf"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	done := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(sourcePath, furtherPath, first, second, third, done)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "").
		Return(mockObjectIterator)

	sc := Client{StorageClient: &storage.Client{}, Oic: mockObjectIteratorCreator, Logger: logging.NewToolLogger("[test]")}
	storageObject, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 2)

	assert.Nil(t, storageObject)
	assert.NotNil(t, err)
}

func TestIsDepthValid(t *testing.T) {
	assert.True(t, isDepthValid(0, "", "object.ovf"))
	assert.True(t, isDepthValid(0, "folder1/folder2", "folder1/folder2/object.ovf"))
	assert.True(t, isDepthValid(1, "folder1", "folder1/folder2/object.ovf"))
	assert.True(t, isDepthValid(2, "", "folder1/folder2/object.ovf"))

	assert.False(t, isDepthValid(0, "", "folder1/object.ovf"))
	assert.False(t, isDepthValid(0, "folder1", "folder1/folder2/object.ovf"))
	assert.False(t, isDepthValid(1, "", "folder1/folder2/object.ovf"))
}

func TestGetBucketNameFromGCSPathObjectInFolderPath(t *testing.T) {
	result, err := GetBucketNameFromGCSPath("gs://bucket_name/folder_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", result)
}

func TestGetBucketNameFromGCSPathObjectPath(t *testing.T) {
	result, err := GetBucketNameFromGCSPath("gs://bucket_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", result)
}

func TestGetBucketNameFromGCSPathBucketOnlyWithTrailingSlash(t *testing.T) {
	result, err := GetBucketNameFromGCSPath("gs://bucket_name/")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", result)
}

func TestGetBucketNameFromGCSPathBucketOnlyWithNoTrailingSlash(t *testing.T) {
	result, err := GetBucketNameFromGCSPath("gs://bucket_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", result)
}

func TestGetBucketNameFromGCSPathBucketErrorWhenNoBucketName(t *testing.T) {
	_, err := GetBucketNameFromGCSPath("gs://")
	assert.NotNil(t, err)
}

func TestGetBucketNameFromGCSPathBucketErrorWhenNoBucketNameTrailingSlash(t *testing.T) {
	_, err := GetBucketNameFromGCSPath("gs:///")
	assert.NotNil(t, err)
}

func TestGetBucketNameFromGCSPathBucketErrorWhenNoBucketNameWithObjectName(t *testing.T) {
	_, err := GetBucketNameFromGCSPath("gs:///object_name")
	assert.NotNil(t, err)
}

func TestGetBucketNameFromGCSPathBucketErrorOnInvalidPath(t *testing.T) {
	_, err := GetBucketNameFromGCSPath("NOT_A_GCS_PATH")
	assert.NotNil(t, err)
}

func TestSplitGCSPathObjectInFolder(t *testing.T) {
	bucket, object, err := SplitGCSPath("gs://bucket_name/folder_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "folder_name/object_name", object)
}

func TestSplitGCSPathObjectDirectlyInBucket(t *testing.T) {
	bucket, object, err := SplitGCSPath("gs://bucket_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "object_name", object)
}

func TestSplitGCSPathBucketOnlyTrailingSlash(t *testing.T) {
	bucket, object, err := SplitGCSPath("gs://bucket_name/")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "", object)
}

func TestSplitGCSPathBucketOnlyNoTrailingSlash(t *testing.T) {
	bucket, object, err := SplitGCSPath("gs://bucket_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "", object)
}

func TestSplitGCSPathObjectNameNonLetters(t *testing.T) {
	bucket, object, err := SplitGCSPath("gs://bucket_name/|||")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "|||", object)
}

func TestSplitGCSPathOErrorOnMissingSlashWhenObjectNameNonLetters(t *testing.T) {
	_, _, err := SplitGCSPath("gs://bucket_name|||")
	assert.NotNil(t, err)
}

func TestSplitGCSPathErrorOnNoBucket(t *testing.T) {
	_, _, err := SplitGCSPath("gs://")
	assert.NotNil(t, err)
}

func TestSplitGCSPathErrorOnNoBucketButObjectPath(t *testing.T) {
	_, _, err := SplitGCSPath("gs:///object_name")
	assert.NotNil(t, err)
}

func TestSplitGCSPathErrorOnInvalidPath(t *testing.T) {
	_, _, err := SplitGCSPath("NOT_A_GCS_PATH")
	assert.NotNil(t, err)
}

func TestGetGCSObjectPathElementsInFolder(t *testing.T) {
	bucket, object, err := GetGCSObjectPathElements("gs://bucket_name/folder_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "folder_name/object_name", object)
}

func TestGetGCSObjectPathElementsNoFolder(t *testing.T) {
	bucket, object, err := GetGCSObjectPathElements("gs://bucket_name/object_name")
	assert.Nil(t, err)
	assert.Equal(t, "bucket_name", bucket)
	assert.Equal(t, "object_name", object)
}

func TestGetGCSObjectPathElementsErrorOnBucketOnlyTrailingSlash(t *testing.T) {
	_, _, err := GetGCSObjectPathElements("gs://bucket_name/")
	assert.NotNil(t, err)
}

func TestGetGCSObjectPathElementsErrorOnBucketOnlyNoTrailingSlash(t *testing.T) {
	_, _, err := GetGCSObjectPathElements("gs://bucket_name")
	assert.NotNil(t, err)
}

func TestGetGCSObjectPathElementsErrorOnInvalidPath(t *testing.T) {
	_, _, err := GetGCSObjectPathElements("NOT_GCS_PATH")
	assert.NotNil(t, err)
}
