//  Copyright 2022 Google Inc. All Rights Reserved.
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
//  limitations under the License

package deleter

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/image"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/stretchr/testify/assert"
)

func TestResourceDeleter_DeletesOnlyFoundImages(t *testing.T) {
	project := "project"
	imageThatExists := image.NewImage(project, "image-1")
	imageThatDoesntExist := image.NewImage(project, "image-2")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(project, imageThatExists.GetImageName()).Return(nil, nil)
	mockCompute.EXPECT().DeleteImage(project, imageThatExists.GetImageName()).Return(nil)
	mockCompute.EXPECT().GetImage(project, imageThatDoesntExist.GetImageName()).Return(nil, errors.New("image not found"))

	deleter := NewResourceDeleter(mockCompute, logging.NewToolLogger("test"))
	deleter.DeleteImagesIfExist([]domain.Image{imageThatExists, imageThatDoesntExist})
}

func TestResourceDeleter_LogsMessage_IfDeleteImagesFails(t *testing.T) {
	project := "project"
	imageThatDeletes := image.NewImage(project, "image-that-deletes")
	imageThatFailsToDelete := image.NewImage(project, "image-that-fails-to-delete")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(project, imageThatDeletes.GetImageName()).Return(nil, nil)
	mockCompute.EXPECT().GetImage(project, imageThatFailsToDelete.GetImageName()).Return(nil, nil)
	mockCompute.EXPECT().DeleteImage(project, imageThatDeletes.GetImageName()).Return(nil)
	mockCompute.EXPECT().DeleteImage(project, imageThatFailsToDelete.GetImageName()).Return(errors.New("delete failed"))
	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().User("Failed to delete \"projects/project/global/images/image-that-fails-to-delete\". Manual deletion required.")
	deleter := NewResourceDeleter(mockCompute, mockLogger)
	deleter.DeleteImagesIfExist([]domain.Image{imageThatDeletes, imageThatFailsToDelete})
}

func TestResourceDeleter_DeletesOnlyFoundDisks(t *testing.T) {
	project := "project"
	zone := "zone"
	diskThatExists, err1 := disk.NewDisk(project, zone, "disk-1")
	diskThatDoesntExist, err2 := disk.NewDisk(project, zone, "disk-2")

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetDisk(diskThatExists.GetProject(), diskThatExists.GetZone(), diskThatExists.GetDiskName()).Return(nil, nil)
	mockCompute.EXPECT().DeleteDisk(diskThatExists.GetProject(), diskThatExists.GetZone(), diskThatExists.GetDiskName()).Return(nil)
	mockCompute.EXPECT().GetDisk(diskThatDoesntExist.GetProject(), diskThatDoesntExist.GetZone(), diskThatDoesntExist.GetDiskName()).Return(nil, errors.New("image not found"))

	deleter := NewResourceDeleter(mockCompute, logging.NewToolLogger("test"))
	deleter.DeleteDisksIfExist([]domain.Disk{diskThatExists, diskThatDoesntExist})
}

func TestResourceDeleter_LogsMessage_IfDeleteDisksFails(t *testing.T) {
	project := "project"
	zone := "zone"
	diskThatDeletes, err1 := disk.NewDisk(project, zone, "disk-that-deletes")
	diskThatFailsToDelete, err2 := disk.NewDisk(project, zone, "disk-that-fails-to-delete")

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetDisk(diskThatDeletes.GetProject(), diskThatDeletes.GetZone(), diskThatDeletes.GetDiskName()).Return(nil, nil)
	mockCompute.EXPECT().GetDisk(diskThatFailsToDelete.GetProject(), diskThatFailsToDelete.GetZone(), diskThatFailsToDelete.GetDiskName()).Return(nil, nil)
	mockCompute.EXPECT().DeleteDisk(diskThatDeletes.GetProject(), diskThatDeletes.GetZone(), diskThatDeletes.GetDiskName()).Return(nil)
	mockCompute.EXPECT().DeleteDisk(diskThatFailsToDelete.GetProject(), diskThatFailsToDelete.GetZone(), diskThatFailsToDelete.GetDiskName()).Return(errors.New("delete failed"))
	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().User("Failed to delete \"projects/project/zones/zone/disks/disk-that-fails-to-delete\". Manual deletion required.")
	deleter := NewResourceDeleter(mockCompute, mockLogger)
	deleter.DeleteDisksIfExist([]domain.Disk{diskThatDeletes, diskThatFailsToDelete})
}
