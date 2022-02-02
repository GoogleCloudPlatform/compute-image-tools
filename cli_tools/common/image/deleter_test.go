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

package image

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestImageDeleter_DeletesOnlyFoundImages(t *testing.T) {
	project := "project"
	imageThatExists := NewImage(project, "image-1")
	imageThatDoesntExist := NewImage(project, "image-2")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(project, imageThatExists.GetImageName()).Return(nil, nil)
	mockCompute.EXPECT().DeleteImage(project, imageThatExists.GetImageName()).Return(nil)
	mockCompute.EXPECT().GetImage(project, imageThatDoesntExist.GetImageName()).Return(nil, errors.New("image not found"))

	deleter := NewImageDeleter(mockCompute, logging.NewToolLogger("test"))
	deleter.DeleteImagesIfExist([]domain.Image{imageThatExists, imageThatDoesntExist})
}

func TestImageDeleter_LogsMessage_IfDeleteFails(t *testing.T) {
	project := "project"
	imageThatDeletes := NewImage(project, "image-that-deletes")
	imageThatFailsToDelete := NewImage(project, "image-that-fails-to-delete")

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
	deleter := NewImageDeleter(mockCompute, mockLogger)
	deleter.DeleteImagesIfExist([]domain.Image{imageThatDeletes, imageThatFailsToDelete})
}
