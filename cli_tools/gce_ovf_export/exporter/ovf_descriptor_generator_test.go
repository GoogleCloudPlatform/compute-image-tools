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
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/compute/v1"
)

func TestOvfDescriptorGenerator_GenerateAndWriteOVFDescriptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	project := "a-project"
	zone := "us-west1-c"
	machineType := "c2-standard-16"
	instanceName := "an-instance"
	machineTypeURI := fmt.Sprintf("projects/%v/zones/%v/machineTypes/%v", project, zone, machineType)
	//TODO
	descriptorStr := "<Envelope><References><File id=\"file0\" href=\"disk1.vmdk\" size=\"0\"></File><File id=\"file1\" href=\"disk2.vmdk\" size=\"0\"></File></References><DiskSection><Info>Virtual disk information</Info><Disk diskId=\"vmdisk0\" fileRef=\"file0\" capacity=\"10\" capacityAllocationUnits=\"byte * 2^30\"></Disk><Disk diskId=\"vmdisk1\" fileRef=\"file1\" capacity=\"20\" capacityAllocationUnits=\"byte * 2^30\"></Disk></DiskSection><VirtualHardwareSection><Info></Info></VirtualHardwareSection><VirtualSystem id=\"\"><Info>A GCE virtual machine</Info><Name>an-instance</Name><OperatingSystemSection id=\"94\" version=\"18\" osType=\"\"><Info>The kind of installed guest operating system</Info><Description>Ubuntu 18.04 (64-bit)</Description></OperatingSystemSection><VirtualHardwareSection><Info>Virtual hardware requirements</Info><System><ElementName>Virtual Hardware Family</ElementName><InstanceID>0</InstanceID><VirtualSystemIdentifier>an-instance</VirtualSystemIdentifier></System><Item><ElementName></ElementName><InstanceID>1</InstanceID><ResourceType>8</ResourceType><Description>SCSI Controller</Description></Item><Item><ElementName>2 virtual CPU(s)</ElementName><InstanceID>2</InstanceID><ResourceType>3</ResourceType><AllocationUnits>hertz * 10^6</AllocationUnits><Description>Number of Virtual CPUs</Description><VirtualQuantity>2</VirtualQuantity></Item><Item><ElementName>2048MB of memory</ElementName><InstanceID>3</InstanceID><ResourceType>4</ResourceType><AllocationUnits>byte * 2^20</AllocationUnits><Description>Memory Size</Description><VirtualQuantity>2048</VirtualQuantity></Item><Item><ElementName>disk0</ElementName><InstanceID>4</InstanceID><ResourceType>17</ResourceType><AddressOnParent>0</AddressOnParent><HostResource>ovf:/disk/vmdisk0</HostResource><Parent>1</Parent></Item><Item><ElementName>disk1</ElementName><InstanceID>5</InstanceID><ResourceType>17</ResourceType><AddressOnParent>1</AddressOnParent><HostResource>ovf:/disk/vmdisk1</HostResource><Parent>1</Parent></Item></VirtualHardwareSection></VirtualSystem></Envelope>"
	bucket := "a-bucket"
	gcsFolder := "folder1/subfolder/"

	mockStorageClient := mocks.NewMockStorageClientInterface(mockCtrl)
	mockStorageClient.EXPECT().WriteToGCS(
		bucket, fmt.Sprintf("%v%v.ovf", gcsFolder, instanceName),
		strings.NewReader(descriptorStr))
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().GetMachineType("a-project", "us-west1-c", machineType).Return(&compute.MachineType{GuestCpus: 2, MemoryMb: 2048, SelfLink: machineTypeURI}, nil)

	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: machineTypeURI,
	}
	disk1 := &ExportedDisk{
		attachedDisk: &compute.AttachedDisk{Boot: true},
		disk:         &compute.Disk{SizeGb: 10},
		gcsPath:      gcsFolder + "disk1.vmdk",
		gcsFileAttrs: &storage.ObjectAttrs{},
	}
	disk2 := &ExportedDisk{
		attachedDisk: &compute.AttachedDisk{},
		disk:         &compute.Disk{SizeGb: 20},
		gcsPath:      gcsFolder + "disk2.vmdk",
		gcsFileAttrs: &storage.ObjectAttrs{},
	}
	exportedDisks := []*ExportedDisk{disk1, disk2}
	diskInspectionResults := &commondisk.InspectionResult{Distro: "ubuntu", Major: "18", Minor: "04", Architecture: "x64"}

	g := ovfDescriptorGeneratorImpl{storageClient: mockStorageClient, computeClient: mockComputeClient, Project: project, Zone: zone}
	err := g.GenerateAndWriteOVFDescriptor(instance, exportedDisks, bucket, gcsFolder, diskInspectionResults)
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
	err := g.GenerateAndWriteOVFDescriptor(instance, []*ExportedDisk{}, "a-bucket", "folder1/subfolder/", nil)
	assert.Equal(t, machineTypeErr, err)
}
