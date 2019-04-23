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

	"github.com/GoogleCloudPlatform/compute-image-tools/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
)

var (
	diskCapacityAllocationUnits = "byte * 2^30"

	fileRef1     = "file1"
	fileRef2     = "file2"
	defaultDisks = &ovf.DiskSection{Disks: []ovf.VirtualDiskDesc{
		{Capacity: "20", CapacityAllocationUnits: &diskCapacityAllocationUnits, DiskID: "vmdisk1", FileRef: &fileRef1},
		{Capacity: "1", CapacityAllocationUnits: &diskCapacityAllocationUnits, DiskID: "vmdisk2", FileRef: &fileRef2},
	}}

	defaultReferences = &[]ovf.File{
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", ID: "file1", Size: 1151322112},
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", ID: "file2", Size: 68096},
	}
)

func TestGetDiskFileInfosDisksOnSingleControllerOutOfOrder(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "1", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "5"),
		},
	}
	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosAllocationUnitExtraSpace(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "1", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "5"),
		},
	}
	extraSpaceDiskCapacityAllocationUnits := "byte * 2^ 30   "
	disks := &ovf.DiskSection{Disks: []ovf.VirtualDiskDesc{
		{Capacity: "11", CapacityAllocationUnits: &extraSpaceDiskCapacityAllocationUnits, DiskID: "vmdisk1", FileRef: &fileRef1},
		{Capacity: "12", CapacityAllocationUnits: &extraSpaceDiskCapacityAllocationUnits, DiskID: "vmdisk2", FileRef: &fileRef2},
	}}

	diskInfos, err := GetDiskInfos(virtualHardware, disks, defaultReferences)

	assert.Nil(t, err)
	assert.NotNil(t, diskInfos)
	assert.Equal(t, 2, len(diskInfos))
	assert.Equal(t, "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", diskInfos[0].FilePath)
	assert.Equal(t, "Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", diskInfos[1].FilePath)
	assert.Equal(t, 11, diskInfos[0].SizeInGB)
	assert.Equal(t, 12, diskInfos[1].SizeInGB)
}

func TestGetDiskFileInfosDisksOnSeparateControllersOutOfOrder(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosInvalidDiskReferenceFormat(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "INVALID_DISK_HOST_RESOURCE", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosMissingDiskReference(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk_DOESNT_EXIST", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosMissingFileReference(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, &[]ovf.File{
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", ID: "file1", Size: 1151322112},
	})
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosDiskWithoutParentController(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
			createDiskItem("8", "0", "disk2", "ovf:/disk/vmdisk3", "123"),
		},
	}

	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosNoControllers(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
			createDiskItem("8", "0", "disk2", "ovf:/disk/vmdisk3", "123"),
		},
	}
	_, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosNilFileReferences(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{}, defaultDisks, nil)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosNilDiskSection(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{}, nil, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosNilDisks(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{}, &ovf.DiskSection{}, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosEmptyDisks(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{},
		&ovf.DiskSection{Disks: []ovf.VirtualDiskDesc{}}, defaultReferences)
	assert.NotNil(t, err)
}

func TestGetDiskFileInfosNilVirtualHardware(t *testing.T) {
	_, err := GetDiskInfos(nil, defaultDisks, defaultReferences)
	assert.NotNil(t, err)
}

func doTestGetDiskFileInfosSuccess(t *testing.T, virtualHardware *ovf.VirtualHardwareSection) {
	diskInfos, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)

	assert.Nil(t, err)
	assert.NotNil(t, diskInfos)
	assert.Equal(t, 2, len(diskInfos))
	assert.Equal(t, "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", diskInfos[0].FilePath)
	assert.Equal(t, "Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", diskInfos[1].FilePath)
	assert.Equal(t, 20, diskInfos[0].SizeInGB)
	assert.Equal(t, 1, diskInfos[1].SizeInGB)
}

func TestGetVirtualHardwareSection(t *testing.T) {
	expected := ovf.VirtualHardwareSection{}
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{expected}}

	virtualHardware, err := GetVirtualHardwareSection(virtualSystem)
	assert.Equal(t, &expected, virtualHardware)
	assert.Nil(t, err)
}

func TestGetVirtualHardwareSectionWhenVirtualSystemNil(t *testing.T) {
	_, err := GetVirtualHardwareSection(nil)
	assert.NotNil(t, err)
}

func TestGetVirtualHardwareSectionWhenVirtualHardwareNil(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: nil}
	_, err := GetVirtualHardwareSection(virtualSystem)
	assert.NotNil(t, err)
}

func TestGetVirtualHardwareSectionWhenVirtualHardwareEmpty(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{}}
	_, err := GetVirtualHardwareSection(virtualSystem)
	assert.NotNil(t, err)
}

func TestGetVirtualSystem(t *testing.T) {
	expected := &ovf.VirtualSystem{}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: expected}
	virtualSystem, err := GetVirtualSystem(ovfDescriptor)

	assert.Equal(t, expected, virtualSystem)
	assert.Nil(t, err)
}

func TestGetVirtualSystemNilOvfDescriptor(t *testing.T) {
	_, err := GetVirtualSystem(nil)
	assert.NotNil(t, err)
}

func TestGetVirtualSystemNilVirtualSystem(t *testing.T) {
	ovfDescriptor := &ovf.Envelope{}
	_, err := GetVirtualSystem(ovfDescriptor)
	assert.NotNil(t, err)
}

func TestGetVirtualHardwareSectionFromDescriptor(t *testing.T) {
	expected := ovf.VirtualHardwareSection{}
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{expected}}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: virtualSystem}

	virtualHardware, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	assert.Equal(t, &expected, virtualHardware)
	assert.Nil(t, err)
}

func TestGetVirtualHardwareSectionFromDescriptorWhenNilVirtualHardware(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: nil}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: virtualSystem}

	_, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	assert.NotNil(t, err)
}

func TestGetVirtualHardwareSectionFromDescriptorWhenNilVirtualSystem(t *testing.T) {
	ovfDescriptor := &ovf.Envelope{VirtualSystem: nil}

	_, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	assert.NotNil(t, err)
}

func TestGetCapacityInGB(t *testing.T) {
	//in GB
	doTestGetCapacityInGB(t, 20, "20", "byte * 2^30")
	doTestGetCapacityInGB(t, 10, "10", "byte * 2^30")
	doTestGetCapacityInGB(t, 1, "1", "byte * 2^30")
	doTestGetCapacityInGB(t, 1024, "1024", "byte * 2^30")
	doTestGetCapacityInGB(t, 5242880, "5242880", "byte * 2^30")

	//in MB
	doTestGetCapacityInGB(t, 1, "1", "byte * 2^20")
	doTestGetCapacityInGB(t, 1, "1024", "byte * 2^20")
	doTestGetCapacityInGB(t, 5*1024, "5242880", "byte * 2^20")

	//in TB
	doTestGetCapacityInGB(t, 1024, "1", "byte * 2^40")
	doTestGetCapacityInGB(t, 5242880*1024, "5242880", "byte * 2^40")
}

func TestGetNumberOfCPUs(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem("1", 3),
		},
	}

	result, err := GetNumberOfCPUs(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(3), result)
}

func TestGetNumberOfCPUsPicksFirst(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createCPUItem("1", 11),
			createCPUItem("2", 2),
			createCPUItem("3", 4),
		},
	}

	result, err := GetNumberOfCPUs(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(11), result)
}

func TestGetNumberOfCPUsErrorWhenVirtualHardwareNil(t *testing.T) {
	_, err := GetNumberOfCPUs(nil)
	assert.NotNil(t, err)
}

func TestGetNumberOfCPUsErrorWhenNoCPUs(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
		},
	}

	_, err := GetNumberOfCPUs(virtualHardware)
	assert.NotNil(t, err)
}

func TestGetMemoryInMB(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createMemoryItem("1", 16),
		},
	}

	result, err := GetMemoryInMB(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(16), result)
}

func TestGetMemoryInMBReturnsFirstMemorySpec(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createMemoryItem("1", 33),
			createMemoryItem("1", 16),
			createMemoryItem("1", 1),
		},
	}

	result, err := GetMemoryInMB(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(33), result)
}

func TestGetMemoryInMBSpecInGB(t *testing.T) {
	virtualHardware := createVirtualHardwareSectionWithMemoryItem(7, "byte * 2^30")
	result, err := GetMemoryInMB(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(7*1024), result)
}

func TestGetMemoryInMBSpecInGBSpacesAroundPower(t *testing.T) {
	virtualHardware := createVirtualHardwareSectionWithMemoryItem(3, "byte * 2^ 30   ")
	result, err := GetMemoryInMB(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(3*1024), result)
}

func TestGetMemoryInMBSpecInTB(t *testing.T) {
	virtualHardware := createVirtualHardwareSectionWithMemoryItem(5, "byte * 2^40")
	result, err := GetMemoryInMB(virtualHardware)
	assert.Nil(t, err)
	assert.Equal(t, int64(5*1024*1024), result)
}

func TestGetMemoryInMBInvalidAllocationUnit(t *testing.T) {
	virtualHardware := createVirtualHardwareSectionWithMemoryItem(5, "NOT_VALID_ALLOCATION_UNIT")
	_, err := GetMemoryInMB(virtualHardware)
	assert.NotNil(t, err)
}

func TestGetMemoryInMBEmptyAllocationUnit(t *testing.T) {
	virtualHardware := createVirtualHardwareSectionWithMemoryItem(5, "")
	_, err := GetMemoryInMB(virtualHardware)
	assert.NotNil(t, err)
}

func TestGetMemoryInMBNilAllocationUnit(t *testing.T) {
	memoryItem := createMemoryItem("1", 33)
	memoryItem.AllocationUnits = nil
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			memoryItem,
		},
	}
	_, err := GetMemoryInMB(virtualHardware)
	assert.NotNil(t, err)
}

func TestGetMemoryInMBReturnsErrorWhenVirtualHardwareNil(t *testing.T) {
	_, err := GetMemoryInMB(nil)
	assert.NotNil(t, err)
}

func TestGetMemoryInMBErrorWhenNoMemory(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("4", usbController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "0", "disk1",
				"ovf:/disk/vmdisk2", "5"),
		},
	}

	_, err := GetMemoryInMB(virtualHardware)
	assert.NotNil(t, err)
}

func TestGetOVFDescriptorAndDiskPaths(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ovfPackagePath := "gs://abucket/apath/"

	virtualHardware := ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", sataController),
			createControllerItem("5", parallelSCSIController),
			createDiskItem("7", "1", "disk1",
				"ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0",
				"ovf:/disk/vmdisk1", "5"),
		},
	}
	ovfDescriptor := &ovf.Envelope{
		Disk:       defaultDisks,
		References: *defaultReferences,
		VirtualSystem: &ovf.VirtualSystem{
			VirtualHardware: []ovf.VirtualHardwareSection{virtualHardware},
		},
	}

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load(ovfPackagePath).Return(ovfDescriptor, nil)

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, ovfPackagePath)
	assert.NotNil(t, ovfDescriptorResult)
	assert.NotNil(t, diskPaths)
	assert.Nil(t, err)

	assert.Equal(t, []DiskInfo{
		{"gs://abucket/apath/Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", 20},
		{"gs://abucket/apath/Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", 1},
	}, diskPaths)
	assert.Equal(t, ovfDescriptor, ovfDescriptorResult)
}

func TestGetOVFDescriptorAndDiskPathsErrorWhenLoadingDescriptor(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load(
		"gs://abucket/apath/").Return(nil, fmt.Errorf("error loading descriptor"))

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, "gs://abucket/apath/")
	assert.Nil(t, ovfDescriptorResult)
	assert.Nil(t, diskPaths)
	assert.NotNil(t, err)
}

func TestGetOVFDescriptorAndDiskPathsErrorWhenNoVirtualSystem(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://abucket/apath/").Return(
		&ovf.Envelope{
			References: *defaultReferences,
			Disk:       defaultDisks,
		}, nil)

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, "gs://abucket/apath/")
	assert.Nil(t, ovfDescriptorResult)
	assert.Nil(t, diskPaths)
	assert.NotNil(t, err)
}

func TestGetOVFDescriptorAndDiskPathsErrorWhenNoVirtualHardware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://abucket/apath/").Return(
		&ovf.Envelope{
			VirtualSystem: &ovf.VirtualSystem{},
			References:    *defaultReferences,
			Disk:          defaultDisks,
		}, nil)

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, "gs://abucket/apath/")
	assert.Nil(t, ovfDescriptorResult)
	assert.Nil(t, diskPaths)
	assert.NotNil(t, err)
}

func TestGetOVFDescriptorAndDiskPathsErrorWhenNoDisks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://abucket/apath/").Return(
		&ovf.Envelope{
			VirtualSystem: &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{
				{Item: []ovf.ResourceAllocationSettingData{
					createControllerItem("3", sataController)},
				},
			}},
			References: *defaultReferences,
		}, nil)

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, "gs://abucket/apath/")
	assert.Nil(t, ovfDescriptorResult)
	assert.Nil(t, diskPaths)
	assert.NotNil(t, err)
}

func TestGetOVFDescriptorAndDiskPathsErrorWhenNoReferences(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockOvfDescriptorLoader := mocks.NewMockOvfDescriptorLoaderInterface(mockCtrl)
	mockOvfDescriptorLoader.EXPECT().Load("gs://abucket/apath/").Return(
		&ovf.Envelope{
			VirtualSystem: &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{
				{Item: []ovf.ResourceAllocationSettingData{createControllerItem("3", sataController)}},
			}},
			Disk: defaultDisks,
		}, nil)

	ovfDescriptorResult, diskPaths, err := GetOVFDescriptorAndDiskPaths(
		mockOvfDescriptorLoader, "gs://abucket/apath/")
	assert.Nil(t, ovfDescriptorResult)
	assert.Nil(t, diskPaths)
	assert.NotNil(t, err)
}

func TestGetOSIdSingleMapping(t *testing.T) {
	osID, err := GetOSId(createOVFDescriptorWithOSType("windows7Server64Guest"))
	assert.Equal(t, "windows-2008r2", osID)
	assert.Nil(t, err)
}

func TestGetOSIdMultiMapping(t *testing.T) {
	osID, err := GetOSId(createOVFDescriptorWithOSType("rhel6_64Guest"))
	assert.Equal(t, "rhel-6", osID)
	assert.Nil(t, err)
}

func TestGetOSIdInvalidOSType(t *testing.T) {
	osID, err := GetOSId(createOVFDescriptorWithOSType("not-an-OS"))
	assert.Equal(t, "", osID)
	assert.NotNil(t, err)
	assert.Equal(t,
		"osType attribute value `not-an-OS` found in OVF descriptor cannot be mapped to an OS supported by Google Compute Engine. Use --os flag to specify OS for this VM",
		err.Error())

}

func TestGetOSIdNonDeterministicSingleOption(t *testing.T) {
	osID, err := GetOSId(createOVFDescriptorWithOSType("ubuntuGuest"))
	assert.Equal(t, "", osID)
	assert.NotNil(t, err)
	assert.Equal(t,
		"cannot determine OS from osType attribute value `ubuntuGuest` found in OVF descriptor. Use --os flag to specify OS for this VM. Potential valid values for given osType attribute are: ubuntu-1404",
		err.Error())
}

func TestGetOSIdNonDeterministicMultiOption(t *testing.T) {
	osID, err := GetOSId(createOVFDescriptorWithOSType("windows8Server64Guest"))
	assert.Equal(t, "", osID)
	assert.NotNil(t, err)
	assert.Equal(t,
		"cannot determine OS from osType attribute value `windows8Server64Guest` found in OVF descriptor. Use --os flag to specify OS for this VM. Potential valid values for given osType attribute are: windows-2012, windows-2012r2",
		err.Error())
}

func createOVFDescriptorWithOSType(osType string) *ovf.Envelope {
	return &ovf.Envelope{
		VirtualSystem: &ovf.VirtualSystem{
			OperatingSystem: []ovf.OperatingSystemSection{
				{
					OSType: &osType,
				},
			},
		},
	}
}

func createVirtualHardwareSectionWithMemoryItem(quantity uint, allocationUnit string) *ovf.VirtualHardwareSection {
	memoryItem := createMemoryItem("1", quantity)
	memoryItem.AllocationUnits = &allocationUnit
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			memoryItem,
		},
	}
	return virtualHardware
}

func doTestGetCapacityInGB(t *testing.T, expected int, capacity string, allocationUnits string) {
	capacityInGB, err := getCapacityInGB(capacity, allocationUnits)
	assert.Nil(t, err)
	assert.Equal(t, expected, capacityInGB)
}

func createControllerItem(instanceID string, resourceType uint16) ovf.ResourceAllocationSettingData {
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:   instanceID,
			ResourceType: &resourceType,
		},
	}
}

func createDiskItem(instanceID string, addressOnParent string,
	elementName string, hostResource string, parent string) ovf.ResourceAllocationSettingData {
	diskType := disk
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &diskType,
			AddressOnParent: &addressOnParent,
			ElementName:     elementName,
			HostResource:    []string{hostResource},
			Parent:          &parent,
		},
	}
}

func createCPUItem(instanceID string, quantity uint) ovf.ResourceAllocationSettingData {
	resourceType := cpu
	mhz := "hertz * 10^6"
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &resourceType,
			VirtualQuantity: &quantity,
			AllocationUnits: &mhz,
		},
	}
}

func createMemoryItem(instanceID string, quantity uint) ovf.ResourceAllocationSettingData {
	resourceType := memory
	mb := "byte * 2^20"

	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceID,
			ResourceType:    &resourceType,
			VirtualQuantity: &quantity,
			AllocationUnits: &mb,
		},
	}
}
