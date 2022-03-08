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

package ovfexporter

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"

	ovfexportdomain "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

func TestOvfDescriptorGenerator_GenerateAndWriteOVFDescriptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	project := "a-project"
	zone := "us-west1-c"
	machineType := "c2-standard-16"
	instanceName := "an-instance"
	descriptorFileName := "descriptor.ovf"
	machineTypeURI := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", project, zone, machineType)
	testTarFileBytes, fileErr := ioutil.ReadFile("../../test_data/ovf_descriptor.ovf")
	assert.NoError(t, fileErr)
	testTarFileString := string(testTarFileBytes)
	assert.NotEmpty(t, testTarFileString)
	bucket := "a-bucket"
	gcsFolder := "folder1/subfolder/"

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().WriteToGCS(
		bucket, fmt.Sprintf("%v%v", gcsFolder, descriptorFileName),
		strings.NewReader(testTarFileString))
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetMachineType("a-project", "us-west1-c", machineType).Return(&compute.MachineType{GuestCpus: 2, MemoryMb: 2048, SelfLink: machineTypeURI}, nil)

	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: machineTypeURI,
	}
	disk1 := &ovfexportdomain.ExportedDisk{
		AttachedDisk: &compute.AttachedDisk{Boot: true},
		Disk:         &compute.Disk{SizeGb: 10},
		GcsPath:      gcsFolder + "disk1.vmdk",
		GcsFileAttrs: &storage.ObjectAttrs{},
	}
	disk2 := &ovfexportdomain.ExportedDisk{
		AttachedDisk: &compute.AttachedDisk{},
		Disk:         &compute.Disk{SizeGb: 20},
		GcsPath:      gcsFolder + "disk2.vmdk",
		GcsFileAttrs: &storage.ObjectAttrs{},
	}
	exportedDisks := []*ovfexportdomain.ExportedDisk{disk1, disk2}
	diskInspectionResults := &pb.InspectionResults{
		OsRelease: &pb.OsRelease{
			Architecture: pb.Architecture_X64,
			Distro:       "ubuntu",
			MajorVersion: "18",
			MinorVersion: "04",
		},
	}
	g := ovfDescriptorGeneratorImpl{storageClient: mockStorageClient, computeClient: mockComputeClient, Project: project, Zone: zone}

	err := g.GenerateAndWriteOVFDescriptor(instance, exportedDisks, bucket, gcsFolder, descriptorFileName, diskInspectionResults)
	assert.Nil(t, err)
}

func TestOvfDescriptorGenerator_GenerateAndWriteOVFDescriptor_ErrorOnGetMachineType(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a-project"
	zone := "us-west1-c"
	machineType := "c2-standard-16"
	machineTypeURI := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", project, zone, machineType)

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	machineTypeErr := fmt.Errorf("machine type error")
	mockComputeClient.EXPECT().GetMachineType("a-project", "us-west1-c", machineType).Return(nil, machineTypeErr)

	instance := &compute.Instance{
		Name:        "an-instance",
		MachineType: machineTypeURI,
	}
	g := ovfDescriptorGeneratorImpl{storageClient: mockStorageClient, computeClient: mockComputeClient, Project: project, Zone: zone}
	err := g.GenerateAndWriteOVFDescriptor(
		instance, []*ovfexportdomain.ExportedDisk{}, "a-bucket",
		"folder1/subfolder/", "descriptor.ovf", nil)
	assert.Equal(t, machineTypeErr, err)
}
