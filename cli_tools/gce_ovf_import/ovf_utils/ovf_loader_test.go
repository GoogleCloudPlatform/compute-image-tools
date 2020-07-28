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
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
)

var (
	ovfPath         = "gs://bucket/folder/descriptor.ovf"
	ovfObjectHandle = &storage.ObjectHandle{}

	infoStr             = "INFO_STR"
	annotationStr       = "ANNOTATION_STR"
	infoSectionRequired = false

	ovfDescriptorStr = fmt.Sprintf(
		"<Envelope><AnnotationSection ovf:required='false'><Info>%v</Info><Annotation>%v</Annotation></AnnotationSection></Envelope>",
		infoStr, annotationStr)
	ovfDescriptor = &ovf.Envelope{
		References: nil,
		Annotation: &ovf.AnnotationSection{
			Section: ovf.Section{
				Required: &infoSectionRequired,
				Info:     infoStr,
			}, Annotation: annotationStr,
		},
	}

	vboxOSDescriptorStr = "<Envelope><VirtualSystem ovf:id=\"vm\"><OperatingSystemSection ovf:id=\"96\"> <Info>The kind of installed guest operating system</Info> <Description>Debian_64</Description> <vbox:OSType ovf:required=\"false\">Debian_64</vbox:OSType></OperatingSystemSection></VirtualSystem></Envelope>"
	vboxOSDescriptor    = &ovf.Envelope{
		References: nil,
		VirtualSystem: &ovf.VirtualSystem{
			Content: ovf.Content{
				ID: "vm",
			},
			OperatingSystem: []ovf.OperatingSystemSection{
				ovf.OperatingSystemSection{
					Section: ovf.Section{
						Info: "The kind of installed guest operating system",
					},
					ID: 96, Description: func(s string) *string { return &s }("Debian_64"),
				},
			},
		},
	}
)

func TestOvfDescriptorLoader(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, ".ovf", 0).Return(ovfObjectHandle, nil).Times(1)
	mockStorageClient.EXPECT().GetGcsFileContent(ovfObjectHandle).Return([]byte(ovfDescriptorStr), nil).Times(1)

	mockOvfDescriptorValidator := mocks.NewMockAbstractOvfDescriptorValidator(mockCtrl)
	mockOvfDescriptorValidator.EXPECT().ValidateOvfPackage(ovfDescriptor, ovfPath).Return(ovfDescriptor, nil).Times(1)

	l := OvfDescriptorLoader{storageClient: mockStorageClient, validator: mockOvfDescriptorValidator}
	result, resultError := l.Load(ovfPath)

	assert.Equal(t, result, ovfDescriptor)
	assert.Nil(t, resultError)
}

func TestOvfDescriptorLoaderNoDescriptorInGcs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	err := fmt.Errorf("no OVF file")
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, ".ovf", 0).Return(nil, err).Times(1)
	mockOvfDescriptorValidator := mocks.NewMockAbstractOvfDescriptorValidator(mockCtrl)

	l := OvfDescriptorLoader{storageClient: mockStorageClient, validator: mockOvfDescriptorValidator}
	result, resultError := l.Load(ovfPath)

	assert.Equal(t, err, resultError)
	assert.Nil(t, result)
}

func TestOvfDescriptorLoaderErrorLoadingDescriptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	err := fmt.Errorf("error loading descriptor")
	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, ".ovf", 0).Return(ovfObjectHandle, nil).Times(1)
	mockStorageClient.EXPECT().GetGcsFileContent(ovfObjectHandle).Return(nil, err).Times(1)
	mockOvfDescriptorValidator := mocks.NewMockAbstractOvfDescriptorValidator(mockCtrl)

	l := OvfDescriptorLoader{storageClient: mockStorageClient, validator: mockOvfDescriptorValidator}
	result, resultError := l.Load(ovfPath)

	assert.Equal(t, err, resultError)
	assert.Nil(t, result)
}

func TestOvfDescriptorLoaderErrorValidatingDescriptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	err := fmt.Errorf("error validating descriptor")

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, ".ovf", 0).Return(ovfObjectHandle, nil).Times(1)
	mockStorageClient.EXPECT().GetGcsFileContent(ovfObjectHandle).Return([]byte(ovfDescriptorStr), nil).Times(1)

	mockOvfDescriptorValidator := mocks.NewMockAbstractOvfDescriptorValidator(mockCtrl)
	mockOvfDescriptorValidator.EXPECT().ValidateOvfPackage(ovfDescriptor, ovfPath).Return(nil, err).Times(1)

	l := OvfDescriptorLoader{storageClient: mockStorageClient, validator: mockOvfDescriptorValidator}
	result, resultError := l.Load(ovfPath)

	assert.Equal(t, err, resultError)
	assert.Nil(t, result)
}

func TestOvfDescriptorLoaderFromVbox(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().FindGcsFileDepthLimited(ovfPath, ".ovf", 0).Return(ovfObjectHandle, nil).Times(1)
	mockStorageClient.EXPECT().GetGcsFileContent(ovfObjectHandle).Return([]byte(vboxOSDescriptorStr), nil).Times(1)

	mockOvfDescriptorValidator := mocks.NewMockAbstractOvfDescriptorValidator(mockCtrl)
	mockOvfDescriptorValidator.EXPECT().ValidateOvfPackage(vboxOSDescriptor, ovfPath).Return(vboxOSDescriptor, nil).Times(1)

	l := OvfDescriptorLoader{storageClient: mockStorageClient, validator: mockOvfDescriptorValidator}
	result, resultError := l.Load(ovfPath)

	assert.Equal(t, result, vboxOSDescriptor)
	assert.Nil(t, resultError)
}
