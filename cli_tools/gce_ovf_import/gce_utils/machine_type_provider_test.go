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
//  limitations under the License

package ovfgceutils

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"
)

func TestGetMachineTypeFromOVFSuccess(t *testing.T) {
	doTestGetMachineTypeSuccess(t, 1, 0.55, "f1-micro")
	doTestGetMachineTypeSuccess(t, 1, 0.6, "f1-micro")
	doTestGetMachineTypeSuccess(t, 1, 0.61, "g1-small")
	doTestGetMachineTypeSuccess(t, 1, 1.7, "g1-small")
	doTestGetMachineTypeSuccess(t, 1, 1.71, "n1-standard-1")
	doTestGetMachineTypeSuccess(t, 1, 3.75, "n1-standard-1")
	doTestGetMachineTypeSuccess(t, 1, 4, "n1-standard-2")
	doTestGetMachineTypeSuccess(t, 1, 3844, "n1-ultramem-160")
	doTestGetMachineTypeSuccess(t, 2, 1.5, "n1-highcpu-2")
	doTestGetMachineTypeSuccess(t, 2, 1.8, "n1-highcpu-2")
	doTestGetMachineTypeSuccess(t, 2, 1.81, "n1-standard-2")
	doTestGetMachineTypeSuccess(t, 2, 7.5, "n1-standard-2")
	doTestGetMachineTypeSuccess(t, 2, 7.51, "n1-highmem-2")
	doTestGetMachineTypeSuccess(t, 2, 416, "n1-ultramem-40")
	doTestGetMachineTypeSuccess(t, 3, 15, "n1-standard-4")
	doTestGetMachineTypeSuccess(t, 4, 14, "n1-standard-4")
	doTestGetMachineTypeSuccess(t, 4, 15, "n1-standard-4")
	doTestGetMachineTypeSuccess(t, 4, 16, "n1-highmem-4")
	doTestGetMachineTypeSuccess(t, 5, 52, "n1-highmem-8")
	doTestGetMachineTypeSuccess(t, 64, 416, "n1-highmem-64")
	doTestGetMachineTypeSuccess(t, 96, 1433.6, "n1-megamem-96")
	doTestGetMachineTypeSuccess(t, 97, 1433.6, "n1-ultramem-160")
	doTestGetMachineTypeSuccess(t, 160, 3844, "n1-ultramem-160")
}

func TestGetMachineTypeTooHighMemoryRequirements(t *testing.T) {
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 1, 3845)
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 2, 3845)
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 161, 3845)
}

func TestGetMachineTypeTooHighCPURequirements(t *testing.T) {
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 161, 1)
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 161, 16)
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 1000, 416)
}

func TestGetMachineTypeTooHighMemoryAndCPURequirements(t *testing.T) {
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 161, 3845)
	doTestGetMachineTypeNoMachineWithCPUMemoryExists(t, 1000, 10000)
}

func TestGetMachineTypeNoVirtualSystemInOVFDescriptor(t *testing.T) {
	mtp := MachineTypeProvider{OvfDescriptor: &ovf.Envelope{}}

	result, err := mtp.GetMachineType()
	assert.NotNil(t, err)
	assert.Equal(t, "", result)
}

func TestGetMachineTypeNoVirtualHardware(t *testing.T) {
	mtp := MachineTypeProvider{OvfDescriptor: &ovf.Envelope{VirtualSystem: &ovf.VirtualSystem{}}}

	result, err := mtp.GetMachineType()
	assert.NotNil(t, err)
	assert.Equal(t, "", result)
}

func TestGetMachineTypeNoCPU(t *testing.T) {
	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createMemoryItem(2048, "1"),
		},
	}
	mtp := MachineTypeProvider{
		OvfDescriptor: &ovf.Envelope{
			VirtualSystem: &ovf.VirtualSystem{
				VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware}}}}

	result, err := mtp.GetMachineType()
	assert.NotNil(t, err)
	assert.Equal(t, "", result)
}

func TestGetMachineTypeNoMemory(t *testing.T) {
	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem(2, "1"),
		},
	}
	mtp := MachineTypeProvider{
		OvfDescriptor: &ovf.Envelope{
			VirtualSystem: &ovf.VirtualSystem{
				VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware}}}}

	result, err := mtp.GetMachineType()
	assert.NotNil(t, err)
	assert.Equal(t, "", result)
}

func TestGetMachineTypeMultipleMemoryItemsPicksFirst(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a_project"
	zone := "us-east1-b"

	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem(2, "1"),
			createMemoryItem(1024, "2"),
			createMemoryItem(4096, "3"),
			createMemoryItem(2048, "4"),
		},
	}
	ovfDescriptor := &ovf.Envelope{
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
		},
	}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, zone).Return(machineTypes, nil).Times(1)

	mtp := MachineTypeProvider{
		OvfDescriptor: ovfDescriptor, ComputeClient: mockComputeClient, Project: project, Zone: zone}

	result, err := mtp.GetMachineType()
	assert.Nil(t, err)
	assert.Equal(t, "n1-highcpu-2", result)
}

func TestGetMachineTypeMultipleCPUItemsPicksFirst(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a_project"
	zone := "us-east1-b"

	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem(16, "1"),
			createCPUItem(2, "2"),
			createCPUItem(8, "3"),
			createMemoryItem(16*1024, "4"),
		},
	}
	ovfDescriptor := &ovf.Envelope{
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
		},
	}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, zone).Return(machineTypes, nil).Times(1)

	mtp := MachineTypeProvider{
		OvfDescriptor: ovfDescriptor, ComputeClient: mockComputeClient, Project: project, Zone: zone}

	result, err := mtp.GetMachineType()
	assert.Nil(t, err)
	assert.Equal(t, "n1-standard-16", result)
}

func TestGetMachineTypeErrorRetrievingMachineTypes(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a_project"
	zone := "us-east1-b"

	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem(2, "1"),
			createMemoryItem(1024, "2"),
		},
	}
	ovfDescriptor := &ovf.Envelope{
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
		},
	}
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, zone).
		Return(nil, fmt.Errorf("can't retrieve machine types")).Times(1)

	mtp := MachineTypeProvider{
		OvfDescriptor: ovfDescriptor, ComputeClient: mockComputeClient, Project: project, Zone: zone}

	result, err := mtp.GetMachineType()
	assert.NotNil(t, err)
	assert.Equal(t, "", result)
}

func TestGetMachineTypeFromMachineTypeFlag(t *testing.T) {
	mtp := MachineTypeProvider{MachineType: "n4-standard-2"}

	result, err := mtp.GetMachineType()
	assert.Nil(t, err)
	assert.Equal(t, "n4-standard-2", result)
}

func doTestGetMachineTypeSuccess(t *testing.T, cpuCount uint, memoryGB float64, expectedMachineType string) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a_project"
	zone := "us-east1-b"

	ovfDescriptor := createCPUAndMemoryOVFDescriptor(cpuCount, uint(memoryGB*1024))
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, zone).Return(machineTypes, nil).Times(1)

	mtp := MachineTypeProvider{
		OvfDescriptor: ovfDescriptor, ComputeClient: mockComputeClient, Project: project, Zone: zone}

	result, err := mtp.GetMachineType()
	assert.Nil(t, err)
	assert.Equal(t, expectedMachineType, result)
}

func doTestGetMachineTypeNoMachineWithCPUMemoryExists(t *testing.T, cpuCount uint, memoryGB float64) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	project := "a_project"
	zone := "us-east1-b"

	ovfDescriptor := createCPUAndMemoryOVFDescriptor(cpuCount, uint(memoryGB*1024))
	mockComputeClient := mocks.NewMockClient(mockCtrl)
	mockComputeClient.EXPECT().ListMachineTypes(project, zone).Return(machineTypes, nil).Times(1)

	mtp := MachineTypeProvider{
		OvfDescriptor: ovfDescriptor, ComputeClient: mockComputeClient, Project: project, Zone: zone}

	result, err := mtp.GetMachineType()
	assert.Equal(t, "", result)
	assert.NotNil(t, err)
	assert.Equal(t, fmt.Sprintf("no machine type has at least %v MBs of memory and %v vCPUs", uint(memoryGB*1024), cpuCount), err.Error())
}

func createCPUAndMemoryOVFDescriptor(cpuCount uint, memoryMB uint) *ovf.Envelope {
	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem(cpuCount, "1"),
			createMemoryItem(memoryMB, "2"),
		},
	}

	return &ovf.Envelope{
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{
				virtualHardware,
			},
		},
	}
}

func createCPUItem(cpuCount uint, id string) ovf.ResourceAllocationSettingData {
	cpuResourceType := uint16(3)
	mhz := "hertz * 10^6"
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      id,
			ResourceType:    &cpuResourceType,
			VirtualQuantity: &cpuCount,
			AllocationUnits: &mhz,
		},
	}
}

func createMemoryItem(memoryMB uint, id string) ovf.ResourceAllocationSettingData {
	memoryResourceType := uint16(4)
	mb := "byte * 2^20"
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      id,
			ResourceType:    &memoryResourceType,
			VirtualQuantity: &memoryMB,
			AllocationUnits: &mb,
		},
	}
}

var machineTypes = []*compute.MachineType{
	{
		GuestCpus:                    1,
		Id:                           1000,
		IsSharedCpu:                  true,
		MaximumPersistentDisks:       16,
		MaximumPersistentDisksSizeGb: 3072,
		MemoryMb:                     614,
		Name:                         "f1-micro",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    1,
		Id:                           2000,
		IsSharedCpu:                  true,
		MaximumPersistentDisks:       16,
		MaximumPersistentDisksSizeGb: 3072,
		MemoryMb:                     1740,
		Name:                         "g1-small",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    16,
		Id:                           4016,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     14746,
		Name:                         "n1-highcpu-16",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    2,
		Id:                           4002,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     1843,
		Name:                         "n1-highcpu-2",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    32,
		Id:                           4032,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     29491,
		Name:                         "n1-highcpu-32",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    4,
		Id:                           4004,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     3686,
		Name:                         "n1-highcpu-4",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    64,
		Id:                           4064,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     58982,
		Name:                         "n1-highcpu-64",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    8,
		Id:                           4008,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     7373,
		Name:                         "n1-highcpu-8",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    96,
		Id:                           4096,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     88474,
		Name:                         "n1-highcpu-96",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    16,
		Id:                           5016,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     106496,
		Name:                         "n1-highmem-16",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    2,
		Id:                           5002,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     13312,
		Name:                         "n1-highmem-2",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    32,
		Id:                           5032,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     212992,
		Name:                         "n1-highmem-32",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    4,
		Id:                           5004,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     26624,
		Name:                         "n1-highmem-4",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    64,
		Id:                           5064,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     425984,
		Name:                         "n1-highmem-64",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    8,
		Id:                           5008,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     53248,
		Name:                         "n1-highmem-8",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    96,
		Id:                           5096,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     638976,
		Name:                         "n1-highmem-96",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    96,
		Id:                           9096,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     1468006,
		Name:                         "n1-megamem-96",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    1,
		Id:                           3001,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     3840,
		Name:                         "n1-standard-1",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    16,
		Id:                           3016,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     61440,
		Name:                         "n1-standard-16",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    2,
		Id:                           3002,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     7680,
		Name:                         "n1-standard-2",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    32,
		Id:                           3032,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     122880,
		Name:                         "n1-standard-32",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    4,
		Id:                           3004,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     15360,
		Name:                         "n1-standard-4",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    64,
		Id:                           3064,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     245760,
		Name:                         "n1-standard-64",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    8,
		Id:                           3008,
		ImageSpaceGb:                 10,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     30720,
		Name:                         "n1-standard-8",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    96,
		Id:                           3096,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     368640,
		Name:                         "n1-standard-96",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    160,
		Id:                           10160,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     3936256,
		Name:                         "n1-ultramem-160",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    40,
		Id:                           10040,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     984064,
		Name:                         "n1-ultramem-40",
		Zone:                         "us-east1-b",
	},
	{
		GuestCpus:                    80,
		Id:                           10080,
		MaximumPersistentDisks:       128,
		MaximumPersistentDisksSizeGb: 65536,
		MemoryMb:                     1968128,
		Name:                         "n1-ultramem-80",
		Zone:                         "us-east1-b",
	},
}
