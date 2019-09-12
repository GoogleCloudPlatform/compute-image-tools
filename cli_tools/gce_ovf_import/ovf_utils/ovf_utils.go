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
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/vmware/govmomi/ovf"
)

const (
	cpu                    uint16 = 3
	memory                 uint16 = 4
	disk                   uint16 = 17
	ideController          uint16 = 5
	parallelSCSIController uint16 = 6
	iSCSIController        uint16 = 8
	sataController         uint16 = 20
	usbController          uint16 = 23
)

//TODO: add Windows 7 and 10 once BYOL is supported
//Full list: https://www.vmware.com/support/developer/vc-sdk/visdk41pubs/ApiReference/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html

// Mapping OVF osType attribute to importer OS ID
var ovfOSTypeToOSID = map[string]string{
	"debian8Guest":          "debian-8",
	"debian8_64Guest":       "debian-8",
	"debian9Guest":          "debian-9",
	"debian9_64Guest":       "debian-9",
	"centos6Guest":          "centos-6",
	"centos6_64Guest":       "centos-6",
	"centos7Guest":          "centos-7",
	"centos7_64Guest":       "centos-7",
	"rhel6Guest":            "rhel-6",
	"rhel6_64Guest":         "rhel-6",
	"rhel7Guest":            "rhel-7",
	"rhel7_64Guest":         "rhel-7",
	"windows7Server64Guest": "windows-2008r2",
}

// Mapping potentially supported OVF osType values to possible importer OS ID values
// Some might have only one option but we can't select it automatically as we cannot guarantee
// correctness.
var noMappingOSTypes = map[string][]string{
	"ubuntuGuest":           {"ubuntu-1404"},
	"ubuntu64Guest":         {"ubuntu-1404", "ubuntu-1604"},
	"windows8Server64Guest": {"windows-2012", "windows-2012r2"},
	"windows9Server64Guest": {"windows-2016"}, //TODO: this will also be Windows 2019 unless VMWare introduces a separate key for it.
}

// DiskInfo holds information about virtual disks in an OVF package
type DiskInfo struct {
	FilePath string
	SizeInGB int
}

// GetDiskInfos returns disk info about disks in a virtual appliance. The first file is boot disk.
func GetDiskInfos(virtualHardware *ovf.VirtualHardwareSection, diskSection *ovf.DiskSection,
	references *[]ovf.File) ([]DiskInfo, error) {
	if virtualHardware == nil {
		return nil, fmt.Errorf("virtualHardware cannot be nil")
	}
	if diskSection == nil || diskSection.Disks == nil || len(diskSection.Disks) == 0 {
		return nil, fmt.Errorf("diskSection cannot be nil")
	}
	if references == nil || *references == nil {
		return nil, fmt.Errorf("references cannot be nil")
	}

	diskControllers := getDiskControllersPrioritized(virtualHardware)
	if len(diskControllers) == 0 {
		return nil, fmt.Errorf("no disk controllers found in OVF, can't retrieve disk info")
	}

	allDiskItems := filterItemsByResourceTypes(virtualHardware, disk)
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
			diskFileName, virtualDiscDesc, err := getDiskFileInfo(
				diskItem.HostResource[0], &diskSection.Disks, references)
			if err != nil {
				return diskInfos, err
			}

			capacityRaw, err := strconv.Atoi(virtualDiscDesc.Capacity)
			if err != nil {
				return diskInfos, err
			}

			allocationUnits := "byte"
			if virtualDiscDesc.CapacityAllocationUnits != nil &&
				*virtualDiscDesc.CapacityAllocationUnits != "" {
				allocationUnits = *virtualDiscDesc.CapacityAllocationUnits
			}
			byteCapacity, err := Parse(int64(capacityRaw), allocationUnits)
			if err != nil {
				return diskInfos, err
			}

			diskInfos = append(diskInfos, DiskInfo{FilePath: diskFileName, SizeInGB: byteCapacity.ToGB()})
		}
	}

	return diskInfos, nil
}

// GetNumberOfCPUs returns number of CPUs in from virtualHardware section. If multiple CPUs are
// defined, the first one will be returned.
func GetNumberOfCPUs(virtualHardware *ovf.VirtualHardwareSection) (int64, error) {
	if virtualHardware == nil {
		return 0, fmt.Errorf("virtualHardware cannot be nil")
	}

	cpuItems := filterItemsByResourceTypes(virtualHardware, cpu)
	if len(cpuItems) == 0 {
		return 0, fmt.Errorf("no CPUs found in OVF")
	}

	// Returning the first CPU item found. Doesn't support multiple deployment configurations.
	return int64(*cpuItems[0].VirtualQuantity), nil
}

// GetMemoryInMB returns memory size in MB from OVF virtualHardware section. If there are multiple
// elements defining memory for the same virtual system, the first memory element will be used.
func GetMemoryInMB(virtualHardware *ovf.VirtualHardwareSection) (int64, error) {
	if virtualHardware == nil {
		return 0, fmt.Errorf("virtualHardware cannot be nil")
	}

	memoryItems := filterItemsByResourceTypes(virtualHardware, memory)
	if len(memoryItems) == 0 {
		return 0, fmt.Errorf("no memory section found in OVF")
	}

	// Using the first memory item found. Doesn't support multiple deployment configurations.
	memoryItem := memoryItems[0]
	if memoryItem.AllocationUnits == nil || *memoryItem.AllocationUnits == "" {
		return 0, fmt.Errorf("memory allocation unit not specified")
	}

	byteCapacity, err := Parse(int64(*memoryItems[0].VirtualQuantity), *memoryItem.AllocationUnits)
	if err != nil {
		return 0, err
	}

	return int64(byteCapacity.ToMB()), nil

}

// GetVirtualHardwareSection returns VirtualHardwareSection from OVF VirtualSystem
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

// GetVirtualSystem returns VirtualSystem element from OVF descriptor envelope
func GetVirtualSystem(ovfDescriptor *ovf.Envelope) (*ovf.VirtualSystem, error) {
	if ovfDescriptor == nil {
		return nil, fmt.Errorf("OVF descriptor is nil, can't extract virtual system")
	}
	if ovfDescriptor.VirtualSystem == nil {
		return nil, fmt.Errorf("OVF descriptor doesn't contain a virtual system")
	}

	return ovfDescriptor.VirtualSystem, nil
}

// GetVirtualHardwareSectionFromDescriptor returns VirtualHardwareSection from OVF descriptor
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

// GetOVFDescriptorAndDiskPaths loads OVF descriptor from GCS folder location. It returns
// descriptor object and full paths to disk files, including ovfGcsPath.
func GetOVFDescriptorAndDiskPaths(ovfDescriptorLoader domain.OvfDescriptorLoaderInterface,
	ovfGcsPath string) (*ovf.Envelope, []DiskInfo, error) {
	ovfDescriptor, err := ovfDescriptorLoader.Load(ovfGcsPath)
	if err != nil {
		return nil, nil, err
	}

	virtualHardware, err := GetVirtualHardwareSectionFromDescriptor(ovfDescriptor)
	if err != nil {
		return nil, nil, err
	}
	diskInfos, err := GetDiskInfos(virtualHardware, ovfDescriptor.Disk, &ovfDescriptor.References)
	if err != nil {
		return nil, nil, err
	}
	for i, d := range diskInfos {
		diskInfos[i].FilePath = ovfGcsPath + d.FilePath
	}
	return ovfDescriptor, diskInfos, nil
}

// GetOSId returns OS ID from OVF descriptor, or error if OS ID could not be retrieved.
func GetOSId(ovfDescriptor *ovf.Envelope) (string, error) {

	if ovfDescriptor.VirtualSystem == nil {
		return "", fmt.Errorf("VirtualSystem must be defined to retrieve OS info")
	}
	if ovfDescriptor.VirtualSystem.OperatingSystem == nil ||
		len(ovfDescriptor.VirtualSystem.OperatingSystem) == 0 {
		return "", fmt.Errorf("OperatingSystemSection must be defined to retrieve OS info")
	}
	var osID string
	var validOSType bool
	osType := *ovfDescriptor.VirtualSystem.OperatingSystem[0].OSType
	if osID, validOSType = ovfOSTypeToOSID[osType]; !validOSType {
		if osIDCandidates, hasOSIDCandidates := noMappingOSTypes[osType]; hasOSIDCandidates {
			return "",
				fmt.Errorf(
					"cannot determine OS from osType attribute value `%v` found in OVF descriptor. Use --os flag to specify OS for this VM. Potential valid values for given osType attribute are: %v",
					osType,
					strings.Join(osIDCandidates, ", "),
				)
		}
		return "", fmt.Errorf("osType attribute value `%v` found in OVF descriptor cannot be mapped to an OS supported by Google Compute Engine. Use --os flag to specify OS for this VM", osType)
	}

	return osID, nil
}

func getDiskControllersPrioritized(virtualHardware *ovf.VirtualHardwareSection) []ovf.ResourceAllocationSettingData {
	controllerItems := filterItemsByResourceTypes(virtualHardware,
		ideController, parallelSCSIController, iSCSIController, sataController, usbController)
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

	diskID, err := extractDiskID(diskHostResource)
	if err != nil {
		return "", nil, err
	}
	for _, disk := range *disks {
		if diskID == disk.DiskID {
			for _, file := range *references {
				if file.ID == *disk.FileRef {
					return file.Href, &disk, nil
				}
			}
			return "", nil, fmt.Errorf("file reference '%v' for disk '%v' not found in OVF descriptor", *disk.FileRef, diskID)
		}
	}
	return "", nil, fmt.Errorf(
		"disk with reference %v couldn't be found in OVF descriptor", diskHostResource)
}

func extractDiskID(diskHostResource string) (string, error) {
	if !strings.HasPrefix(diskHostResource, "ovf:/disk/") {
		return "", fmt.Errorf("disk host resource %v has invalid format", diskHostResource)
	}
	return strings.TrimPrefix(diskHostResource, "ovf:/disk/"), nil
}

func sortItemsByStringValue(items []ovf.ResourceAllocationSettingData, extractValue func(ovf.ResourceAllocationSettingData) string) {
	sort.SliceStable(items, func(i, j int) bool {
		iVal := extractValue(items[i])
		jVal := extractValue(items[j])
		iInstanceID, iErr := strconv.Atoi(iVal)
		jInstanceID, jErr := strconv.Atoi(jVal)
		if iErr == nil && jErr == nil {
			return iInstanceID < jInstanceID
		}
		return strings.Compare(iVal, jVal) == -1
	})
}
