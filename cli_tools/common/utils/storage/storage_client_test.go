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
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
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

	sc := StorageClient{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter,
		Logger: logging.NewLogger("[test]")}
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
	mockObjectIteratorCreator.EXPECT().CreateObjectIterator(
		"sourcebucket", "sourcepath/furtherpath").Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
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

	sc := StorageClient{Oic: mockObjectIteratorCreator, ObjectDeleter: mockStorageObjectDeleter,
		Logger: logging.NewLogger("[test]")}
	err := sc.DeleteGcsPath("gs://sourcebucket/sourcepath/furtherpath")
	assert.NotNil(t, err)
}

func TestFindGcsFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/bingo.ovf"}, nil)
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.FindGcsFile(
		"gs://sourcebucket/sourcepath/furtherpath", ".ovf")

	assert.NotNil(t, objectHandle)
	assert.Equal(t, "sourcebucket", objectHandle.BucketName())
	assert.Equal(t, "sourcepath/furtherpath/bingo.ovf", objectHandle.ObjectName())
	assert.Nil(t, err)
}

func TestFindGcsFileNoFileFound(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockObjectIterator := mocks.NewMockObjectIteratorInterface(mockCtrl)
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile4"}, nil)
	fourth := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile5.txt"}, nil)
	fifth := mockObjectIterator.EXPECT().Next().Return(nil, iterator.Done)
	gomock.InOrder(first, second, third, fourth, fifth)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
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
	first := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile1.txt"}, nil)
	second := mockObjectIterator.EXPECT().Next().
		Return(&storage.ObjectAttrs{Name: "sourcepath/furtherpath/afile2.txt"}, nil)
	third := mockObjectIterator.EXPECT().Next().
		Return(nil, fmt.Errorf("error while iterating"))
	gomock.InOrder(first, second, third)

	mockObjectIteratorCreator := mocks.NewMockObjectIteratorCreatorInterface(mockCtrl)
	mockObjectIteratorCreator.EXPECT().
		CreateObjectIterator("sourcebucket", "sourcepath/furtherpath").
		Return(mockObjectIterator)

	sc := StorageClient{Oic: mockObjectIteratorCreator, Logger: logging.NewLogger("[test]")}
	objectHandle, err := sc.
		FindGcsFile("gs://sourcebucket/sourcepath/furtherpath", ".ovf")
	assert.Nil(t, objectHandle)
	assert.NotNil(t, err)
}
