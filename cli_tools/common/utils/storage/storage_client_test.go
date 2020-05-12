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
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"
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

	mockStorageObjectDeleter := mocks.NewMockStorageObjectDeleterInterface(mockCtrl)
	firstDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile1.txt")
	secondDeletion := mockStorageObjectDeleter.EXPECT().DeleteObject("sourcebucket", "sourcepath/furtherpath/afile2.txt")
	gomock.InOrder(firstDeletion, secondDeletion)

	sc := Client{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter,
		Logger: logging.NewLogger("[test]")}
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
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

	mockStorageObjectDeleter := mocks.NewMockStorageObjectDeleterInterface(mockCtrl)
	firstDeletion := mockStorageObjectDeleter.EXPECT().
		DeleteObject("sourcebucket", "sourcepath/furtherpath/afile1.txt").
		Return(nil)
	secondDeletion := mockStorageObjectDeleter.EXPECT().
		DeleteObject("sourcebucket", "sourcepath/furtherpath/afile2.txt").
		Return(fmt.Errorf("can't delete second file"))
	gomock.InOrder(firstDeletion, secondDeletion)

	sc := Client{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter,
		Logger: logging.NewLogger("[test]")}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFile(
		fmt.Sprintf("gs://%v/%v", bucket, lookupPath), ".ovf")

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", objectHandle.ObjectName())
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, objectHandle)
	assert.NotNil(t, err)
}

func TestFindGcsFileInvalidGCSPath(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	sc := Client{}
	objectHandle, err := sc.FindGcsFile("NOT_A_GCS_PATH", ".ovf")
	assert.Nil(t, objectHandle)
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.
		FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, objectHandle)
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 0)

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "bingo.ovf", objectHandle.ObjectName())
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 0)

	assert.Nil(t, objectHandle)
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFileDepthLimited(
		gcsDirectoryPath, ".ovf", 2)

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", objectHandle.ObjectName())
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFileDepthLimited(
		fmt.Sprintf("gs://%v/%v", bucket, lookupPath), ".ovf", 0)

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", objectHandle.ObjectName())
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

	sc := Client{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFileDepthLimited(
		"gs://sourcebucket/", ".ovf", 2)

	assert.Nil(t, objectHandle)
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
