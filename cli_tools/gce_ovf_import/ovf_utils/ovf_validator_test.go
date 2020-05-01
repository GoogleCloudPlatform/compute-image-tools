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

package ovfutils

import (
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
)

var (
	ovfPathForValidation = "gs://bucket/folder/descriptor.ovf"
)

func TestValidateOvfPackage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	references := []ovf.File{file(1), file(2), file(3)}
	ovfDescriptorForValidation := envelope(references)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	for _, reference := range references {
		mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, reference.Href, 0).Return(&storage.ObjectHandle{}, nil).Times(1)
	}

	v := OvfValidator{mockStorageClient}
	result, resultError := v.ValidateOvfPackage(ovfDescriptorForValidation, ovfPathForValidation)

	assert.Equal(t, result, ovfDescriptorForValidation)
	assert.Nil(t, resultError)
}

func TestValidateOvfPackageWhenReferencesNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ovfDescriptorForValidation := envelope(nil)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	v := OvfValidator{mockStorageClient}
	result, resultError := v.ValidateOvfPackage(ovfDescriptorForValidation, ovfPathForValidation)

	assert.Equal(t, result, ovfDescriptorForValidation)
	assert.Nil(t, resultError)
}

func TestValidateOvfPackageErrorWhenDescriptorNil(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	v := OvfValidator{mockStorageClient}
	result, resultError := v.ValidateOvfPackage(nil, ovfPathForValidation)

	assert.NotNil(t, resultError)
	assert.Nil(t, result)
}

func TestValidateOvfPackageMissingMiddleReferenceInGcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	err := fmt.Errorf("no file found")
	references := []ovf.File{file(1), file(2), file(3)}
	ovfDescriptorForValidation := envelope(references)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, references[0].Href, 0).Return(&storage.ObjectHandle{}, nil).Times(1)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, references[1].Href, 0).Return(nil, err).Times(1)

	v := OvfValidator{mockStorageClient}
	result, resultError := v.ValidateOvfPackage(ovfDescriptorForValidation, ovfPathForValidation)

	assert.NotNil(t, resultError)
	assert.Nil(t, result)
}

func TestValidateOvfPackageMissingFirstReferenceInGcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	err := fmt.Errorf("no file found")
	references := []ovf.File{file(1), file(2), file(3)}
	ovfDescriptorForValidation := envelope(references)
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)

	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, references[0].Href, 0).Return(nil, err).Times(1)

	v := OvfValidator{mockStorageClient}
	result, resultError := v.ValidateOvfPackage(ovfDescriptorForValidation, ovfPathForValidation)

	assert.NotNil(t, resultError)
	assert.Nil(t, result)
}

func file(index int) ovf.File {
	return ovf.File{
		ID:   fmt.Sprintf("id%v", index),
		Href: fmt.Sprintf("ref%v", index),
		Size: 1,
	}
}

func envelope(references []ovf.File) *ovf.Envelope {
	return &ovf.Envelope{References: references}
}
