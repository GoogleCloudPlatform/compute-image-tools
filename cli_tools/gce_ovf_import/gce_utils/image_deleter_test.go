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

package ovfgceutils

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	ovfdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
)

func TestImageDeleter_DeletesImageIfFound(t *testing.T) {
	project := "project"
	imageThatExists := ovfdomain.NewImage(project, "image-1")
	imageThatDoesntExist := ovfdomain.NewImage(project, "image-2")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(project, imageThatExists.ImageName).Return(nil, nil)
	mockCompute.EXPECT().DeleteImage(project, imageThatExists.ImageName).Return(nil)
	mockCompute.EXPECT().GetImage(project, imageThatDoesntExist.ImageName).Return(nil, errors.New("image not found"))

	deleter := NewImageDeleter(mockCompute, logging.NewToolLogger("test"))
	deleter.DeleteImagesIfExist([]ovfdomain.Image{imageThatExists, imageThatDoesntExist})
}

func TestImageDeleter_LogsMessage_IfDeleteFails(t *testing.T) {
	project := "project"
	img := ovfdomain.NewImage(project, "image-1")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCompute := mocks.NewMockClient(ctrl)
	mockCompute.EXPECT().GetImage(project, img.ImageName).Return(nil, nil)
	mockCompute.EXPECT().DeleteImage(project, img.ImageName).Return(errors.New("delete failed"))

	mockLogger := mocks.NewMockToolLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any()).AnyTimes()
	mockLogger.EXPECT().User("Failed to delete \"projects/project/global/images/image-1\". Manual deletion required.")
	deleter := NewImageDeleter(mockCompute, mockLogger)
	deleter.DeleteImagesIfExist([]ovfdomain.Image{img})
}
