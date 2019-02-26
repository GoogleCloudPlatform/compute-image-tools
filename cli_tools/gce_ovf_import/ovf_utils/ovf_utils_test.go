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
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/ovf"
	"reflect"
	"testing"
)

var (
	capacityAllocationUnits = "byte * 2^30"

	fileRef1     = "file1"
	fileRef2     = "file2"
	defaultDisks = &[]ovf.VirtualDiskDesc{
		{Capacity: "20", CapacityAllocationUnits: &capacityAllocationUnits, DiskID: "vmdisk1", FileRef: &fileRef1},
		{Capacity: "1", CapacityAllocationUnits: &capacityAllocationUnits, DiskID: "vmdisk2", FileRef: &fileRef2},
	}

	defaultReferences = &[]ovf.File{
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", ID: "file1", Size: 1151322112},
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk2.vmdk", ID: "file2", Size: 68096},
	}
)

func TestGetDiskFileInfosDisksOnSingleControllerOutOfOrder(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "1", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "5"),
		},
	}
	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosDisksOnSeparateControllersOutOfOrder(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosInvalidDiskReferenceFormat(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "INVALID_DISK_HOST_RESOURCE", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)
	if err == nil {
		t.Error("error expected", err)
	}
}

func TestGetDiskFileInfosMissingDiskReference(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk_DOESNT_EXIST", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, defaultReferences)
	if err == nil {
		t.Error("error expected", err)
	}
}

func TestGetDiskFileInfosMissingFileReference(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
		},
	}

	_, err := GetDiskInfos(virtualHardware, defaultDisks, &[]ovf.File{
		{Href: "Ubuntu_for_Horizon71_1_1.0-disk1.vmdk", ID: "file1", Size: 1151322112},
	})
	if err == nil {
		t.Error("error expected", err)
	}
}

func TestGetDiskFileInfosDiskWithoutParentController(t *testing.T) {
	virtualHardware := &ovf.VirtualHardwareSection{
		Item: []ovf.ResourceAllocationSettingData{
			createControllerItem("3", SATAController),
			createControllerItem("4", USBController),
			createControllerItem("5", ParallelSCSIController),
			createDiskItem("7", "0", "disk1", "ovf:/disk/vmdisk2", "5"),
			createDiskItem("6", "0", "disk0", "ovf:/disk/vmdisk1", "3"),
			createDiskItem("8", "0", "disk2", "ovf:/disk/vmdisk3", "123"),
		},
	}

	doTestGetDiskFileInfosSuccess(t, virtualHardware)
}

func TestGetDiskFileInfosNilFileReferences(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{}, defaultDisks, nil)
	if err == nil {
		t.Error("error expected", err)
	}
}

func TestGetDiskFileInfosNilDisks(t *testing.T) {
	_, err := GetDiskInfos(&ovf.VirtualHardwareSection{}, nil, defaultReferences)
	if err == nil {
		t.Error("error expected", err)
	}
}

func TestGetDiskFileInfosNilVirtualHardware(t *testing.T) {
	_, err := GetDiskInfos(nil, defaultDisks, defaultReferences)
	if err == nil {
		t.Error("error expected", err)
	}
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
	if !reflect.DeepEqual(*virtualHardware, expected) {
		t.Errorf("%v returned for virtual system when %v expected", virtualSystem, expected)
	}
	if err != nil {
		t.Errorf("%v error returned for virtual system when nil expected", err)
	}
}

func TestGetVirtualHardwareSectionWhenVirtualSystemNil(t *testing.T) {
	_, err := GetVirtualHardwareSection(nil)
	if err == nil {
		t.Error("nil error returned when virtual system is nil", err)
	}
}

func TestGetVirtualHardwareSectionWhenVirtualHardwareNil(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: nil}
	_, err := GetVirtualHardwareSection(virtualSystem)
	if err == nil {
		t.Error("nil error returned when virtual hardware slice is nil", err)
	}
}

func TestGetVirtualHardwareSectionWhenVirtualHardwareEmpty(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{}}
	_, err := GetVirtualHardwareSection(virtualSystem)
	if err == nil {
		t.Error("nil error returned when virtual hardware slice is empty", err)
	}
}

func TestGetVirtualSystem(t *testing.T) {
	expected := &ovf.VirtualSystem{}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: expected}
	virtualSystem, err := GetVirtualSystem(ovfDescriptor)

	if virtualSystem != expected {
		t.Errorf("%v returned for virtual system when %v expected", virtualSystem, expected)
	}
	if err != nil {
		t.Errorf("%v error returned for virtual system when nil expected", err)
	}
}

func TestGetVirtualSystemNilOvfDescriptor(t *testing.T) {
	_, err := GetVirtualSystem(nil)

	if err == nil {
		t.Error("nil error returned when OVF descriptor is nil")
	}
}

func TestGetVirtualSystemNilVirtualSystem(t *testing.T) {
	ovfDescriptor := &ovf.Envelope{}
	_, err := GetVirtualSystem(ovfDescriptor)

	if err == nil {
		t.Errorf("nil error returned when OVF descriptor is nil")
	}
}

func TestGetVirtualHardwareSectionFromDescriptor(t *testing.T) {
	expected := ovf.VirtualHardwareSection{}
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: []ovf.VirtualHardwareSection{expected}}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: virtualSystem}

	virtualHardware, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	if !reflect.DeepEqual(*virtualHardware, expected) {
		t.Errorf("%v returned for virtual system when %v expected", virtualSystem, expected)
	}
	if err != nil {
		t.Errorf("%v error returned when retrieving virtual hardware, but nil error expected", err)
	}
}

func TestGetVirtualHardwareSectionFromDescriptorWhenNilVirtualHardware(t *testing.T) {
	virtualSystem := &ovf.VirtualSystem{VirtualHardware: nil}
	ovfDescriptor := &ovf.Envelope{VirtualSystem: virtualSystem}

	_, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	if err == nil {
		t.Errorf("nil error returned when virtual hardware is nil")
	}
}

func TestGetVirtualHardwareSectionFromDescriptorWhenNilVirtualSystem(t *testing.T) {
	ovfDescriptor := &ovf.Envelope{VirtualSystem: nil}

	_, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	if err == nil {
		t.Errorf("nil error returned when virtual system is nil")
	}
}

func createControllerItem(instanceId string, resourceType uint16) ovf.ResourceAllocationSettingData {
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:   instanceId,
			ResourceType: &resourceType,
		},
		Required:      nil,
		Configuration: nil,
		Bound:         nil,
	}
}

func createDiskItem(instanceId string, addressOnParent string,
	elementName string, hostResource string, parent string) ovf.ResourceAllocationSettingData {
	diskType := Disk
	return ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			InstanceID:      instanceId,
			ResourceType:    &diskType,
			AddressOnParent: &addressOnParent,
			ElementName:     elementName,
			HostResource:    []string{hostResource},
			Parent:          &parent,
		},
		Required:      nil,
		Configuration: nil,
		Bound:         nil,
	}
}
