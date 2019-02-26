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
	"github.com/vmware/govmomi/ovf"
	"sort"
	"strconv"
	"strings"
)

const (
	Disk                   uint16 = 17
	IDEController          uint16 = 5
	ParallelSCSIController uint16 = 6
	ISCSIController        uint16 = 8
	SATAController         uint16 = 20
	USBController          uint16 = 23
)

type DiskInfo struct {
	FilePath string
	SizeInGB int
}

// Returns file names of disks in the virtual appliance. The first file is boot disk.
func GetDiskInfos(virtualHardware *ovf.VirtualHardwareSection, disks *[]ovf.VirtualDiskDesc,
	references *[]ovf.File) ([]DiskInfo, error) {
	if virtualHardware == nil {
		return nil, fmt.Errorf("virtualHardware cannot be nil")
	}
	if disks == nil {
		return nil, fmt.Errorf("disks cannot be nil")
	}
	if references == nil {
		return nil, fmt.Errorf("references cannot be nil")
	}

	diskControllers := getDiskControllersPrioritized(virtualHardware)
	allDiskItems := filterItemsByResourceTypes(virtualHardware, Disk)
	diskInfos := make([]DiskInfo, 0)

	for _, diskController := range diskControllers {
		controllerDisks := make([]ovf.ResourceAllocationSettingData, 0)

		for _, diskItem := range allDiskItems {
			if *diskItem.Parent == diskController.InstanceID {
				controllerDisks = append(controllerDisks, diskItem)
			}
		}

		sortItemsByStringValue(controllerDisks, func(disk ovf.ResourceAllocationSettingData) string {
			return *disk.AddressOnParent
		})

		for _, diskItem := range controllerDisks {
			if diskFileName, virtualDiscDesc, err := getDiskFileInfo(diskItem.HostResource[0], disks, references); err != nil {
				return diskInfos, err
			} else {
				diskInfo := DiskInfo{FilePath: diskFileName}
				sizeInGB, err := strconv.Atoi(virtualDiscDesc.Capacity)
				if err == nil {
					//TODO: parse allocation unit and round up to GB
					if *virtualDiscDesc.CapacityAllocationUnits != "byte * 2^30" {
						return nil, fmt.Errorf("support for %v units not implemented", *virtualDiscDesc.CapacityAllocationUnits)
					}
					diskInfo.SizeInGB = sizeInGB
				}

				diskInfos = append(diskInfos, diskInfo)
			}
		}
	}

	return diskInfos, nil
}

func GetVirtualHardwareSection(virtualSystem *ovf.VirtualSystem) (*ovf.VirtualHardwareSection, error) {
	//TODO: support for multiple VirtualHardwareSection for different environments
	//More on page 50, https://www.dmtf.org/sites/default/files/standards/documents/DSP2017_2.0.0.pdf
	if virtualSystem == nil {
		return nil, fmt.Errorf("virtual system is nil, can't extract Virtual hardware")
	}
	if virtualSystem.VirtualHardware == nil || len(virtualSystem.VirtualHardware) == 0 {
		return nil, fmt.Errorf("virtual hardware is nil or empty")
	}
	return &virtualSystem.VirtualHardware[0], nil
}

func GetVirtualSystem(ovfDescriptor *ovf.Envelope) (*ovf.VirtualSystem, error) {
	if ovfDescriptor == nil {
		return nil, fmt.Errorf("OVF descriptor is nil, can't extract virtual system")
	}
	if ovfDescriptor.VirtualSystem == nil {
		return nil, fmt.Errorf("OVF descriptor doesn't contain a virtual system")
	}

	return ovfDescriptor.VirtualSystem, nil
}

func GetVirtualHardwareSectionFromDescriptor(ovfDescriptor *ovf.Envelope) (*ovf.VirtualHardwareSection, error) {
	virtualSystem, err := GetVirtualSystem(ovfDescriptor)
	if err != nil {
		return nil, err
	}
	virtualHardware, err := GetVirtualHardwareSection(virtualSystem)
	if err != nil {
		return nil, err
	}
	return virtualHardware, nil
}

func getDiskControllersPrioritized(virtualHardware *ovf.VirtualHardwareSection) []ovf.ResourceAllocationSettingData {
	controllerItems := filterItemsByResourceTypes(virtualHardware,
		IDEController, ParallelSCSIController, ISCSIController, SATAController, USBController)
	sortItemsByStringValue(controllerItems, func(item ovf.ResourceAllocationSettingData) string {
		return item.InstanceID
	})
	return controllerItems
}

func filterItemsByResourceTypes(virtualHardware *ovf.VirtualHardwareSection, resourceTypes ...uint16) []ovf.ResourceAllocationSettingData {
	filtered := make([]ovf.ResourceAllocationSettingData, 0)
	for _, item := range virtualHardware.Item {
		for _, resourceType := range resourceTypes {
			if *item.ResourceType == resourceType {
				filtered = append(filtered, item)
			}
		}
	}
	return filtered
}

func getDiskFileInfo(diskHostResource string, disks *[]ovf.VirtualDiskDesc,
	references *[]ovf.File) (string, *ovf.VirtualDiskDesc, error) {

	diskId, err := extractDiskId(diskHostResource)
	if err != nil {
		return "", nil, err
	}
	for _, disk := range *disks {
		if diskId == disk.DiskID {
			for _, file := range *references {
				if file.ID == *disk.FileRef {
					return file.Href, &disk, nil
				}
			}
			return "", nil, fmt.Errorf("file reference '%v' for disk '%v' not found in OVF descriptor", *disk.FileRef, diskId)
		}
	}
	return "", nil, fmt.Errorf(
		"disk with reference %v couldn't be found in OVF descriptor", diskHostResource)
}

func extractDiskId(diskHostResource string) (string, error) {
	if !strings.HasPrefix(diskHostResource, "ovf:/disk/") {
		return "", fmt.Errorf("disk host resource %v has invalid format", diskHostResource)
	}
	return strings.TrimPrefix(diskHostResource, "ovf:/disk/"), nil
}

func sortItemsByStringValue(items []ovf.ResourceAllocationSettingData, extractValue func(ovf.ResourceAllocationSettingData) string) {
	sort.SliceStable(items, func(i, j int) bool {
		iVal := extractValue(items[i])
		jVal := extractValue(items[j])
		iInstanceId, iErr := strconv.Atoi(iVal)
		jInstanceId, jErr := strconv.Atoi(jVal)
		if iErr == nil && jErr == nil {
			return iInstanceId < jInstanceId
		}
		return strings.Compare(iVal, jVal) == -1
	})
}
