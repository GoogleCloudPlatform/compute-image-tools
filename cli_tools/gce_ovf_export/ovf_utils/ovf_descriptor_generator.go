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

package ovfutils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	daisycompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"
)

//TODO: merge with OVF import ovf_utils.go
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

// OvfDescriptorGenerator is responsible for generating OVF descriptor based on GCE instance being exported
type OvfDescriptorGenerator struct {
	ComputeClient daisycompute.Client
	StorageClient domain.StorageClientInterface
	Project       string
	Zone          string
}

// Load finds and loads OVF descriptor from a GCS directory path.
// ovfGcsPath is a path to OVF directory, not a path to OVF descriptor file itself.
func (g *OvfDescriptorGenerator) Generate(instance *compute.Instance, exportedDisksGCSPaths []string) (*ovf.Envelope, error) {
	instanceID := "0"
	descriptor := &ovf.Envelope{}
	descriptor.VirtualHardware = &ovf.VirtualHardwareSection{}
	descriptor.References = make([]ovf.File, len(exportedDisksGCSPaths))
	descriptor.Disk = &ovf.DiskSection{Disks: make([]ovf.VirtualDiskDesc, len(exportedDisksGCSPaths)), Section: ovf.Section{Info: "Virtual disk information"}}

	for diskIndex, diskGCSPath := range exportedDisksGCSPaths {
		diskGCSFileName := diskGCSPath
		if slash := strings.LastIndex(diskGCSFileName, "/"); slash > -1 {
			diskGCSFileName = diskGCSFileName[slash+1:]
		}
		//TODO: disk file sizes
		descriptor.References[diskIndex] = ovf.File{Href: diskGCSFileName, ID: "file" + strconv.Itoa(diskIndex), Size: 0}

		//TODO; capacity,CapacityAllocationUnits, PopulatedSize
		descriptor.Disk.Disks[diskIndex] = ovf.VirtualDiskDesc{DiskID: "vmdisk" + strconv.Itoa(diskIndex), FileRef: &descriptor.References[diskIndex].Href, Capacity: "0", CapacityAllocationUnits: nil, PopulatedSize: nil}

		//TODO: parentID
		descriptor.VirtualSystem.VirtualHardware[0].Item = append(
			descriptor.VirtualSystem.VirtualHardware[0].Item,
			*g.createDiskItem(strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item)+1), "1", diskIndex))
	}
	//TODO: virtual hardware section for ordering? might not be necessary for export. We used it for import to determine order of disks if disks are spread over multiple controllers

	descriptor.VirtualSystem = &ovf.VirtualSystem{Content: ovf.Content{Info: "A GCE virtual machine", Name: &instance.Name}}
	descriptor.VirtualSystem.VirtualHardware = make([]ovf.VirtualHardwareSection, 1)
	descriptor.VirtualSystem.VirtualHardware[0] = ovf.VirtualHardwareSection{Section: ovf.Section{Info: "Virtual hardware requirements"}}

	//TODO: operating system

	//TODO: do we need VirtualSystemType? It's set to e.g. "vmx-11". It's a VMWare identifier
	descriptor.VirtualSystem.VirtualHardware[0].System = &ovf.VirtualSystemSettingData{CIMVirtualSystemSettingData: ovf.CIMVirtualSystemSettingData{ElementName: "Virtual Hardware Family", InstanceID: instanceID, VirtualSystemIdentifier: &instance.Name}}

	//TODO items: network,  video card, disks, scsi/sata controllers

	machineType, err := g.ComputeClient.GetMachineType(g.Project, g.Zone, instance.MachineType)
	if err != nil {
		return descriptor, err
	}
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item,
		*g.createCPUItem(machineType, strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item)+1)))
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item,
		*g.createMemoryItem(machineType, strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item)+1)))

	return descriptor, nil
}

func appendItem(item *ovf.ResourceAllocationSettingData, items *[]ovf.ResourceAllocationSettingData) {
	*items = append(*items, *item)
}

func (g *OvfDescriptorGenerator) createDiskItem(instanceID string, parentID string, addressOnParent int) *ovf.ResourceAllocationSettingData {
	return &ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			AddressOnParent: strPtr(strconv.Itoa(addressOnParent)),
			ElementName:     fmt.Sprintf("disk%v", addressOnParent), //TODO
			HostResource:    []string{"ovf:/disk/vmdisk2"},          //TODO
			InstanceID:      instanceID,
			Parent:          strPtr(parentID),
			ResourceType:    func() *uint16 { v := disk; return &v }(),
		},
	}
}

func (g *OvfDescriptorGenerator) createCPUItem(machineType *compute.MachineType, instanceID string) *ovf.ResourceAllocationSettingData {
	return &ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			AllocationUnits: strPtr("hertz * 10^6"),
			Description:     strPtr("Number of Virtual CPUs"),
			ElementName:     fmt.Sprintf("%v virtual CPU(s)", machineType.GuestCpus),
			InstanceID:      instanceID,
			ResourceType:    func() *uint16 { v := cpu; return &v }(),
			//TODO: information loss possible only on 32-bit systems, highly unlikely to happen
			VirtualQuantity: func() *uint { v := uint(machineType.GuestCpus); return &v }(),
		},
	}
}

func (g *OvfDescriptorGenerator) createMemoryItem(machineType *compute.MachineType, instanceID string) *ovf.ResourceAllocationSettingData {
	return &ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			AllocationUnits: strPtr("byte * 2^20"),
			Description:     strPtr("Memory Size"),
			ElementName:     fmt.Sprintf("%vMB of memory", machineType.MemoryMb),
			InstanceID:      instanceID,
			ResourceType:    func() *uint16 { v := memory; return &v }(),
			//TODO: information loss possible only on 32-bit systems, highly unlikely to happen
			VirtualQuantity: func() *uint { v := uint(machineType.MemoryMb); return &v }(),
		},
	}
}

func marshal(descriptor *ovf.Envelope) (string, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	err := enc.Encode(&descriptor)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func strPtr(s string) *string {
	return &s
}

func (g *OvfDescriptorGenerator) GenerateAndWriteOVFDescriptor(instance *compute.Instance, bucketName, gcsDirectoryPath string, exportedDisksGCSPaths []string) error {
	var err error
	var descriptor *ovf.Envelope
	if descriptor, err = g.Generate(instance, exportedDisksGCSPaths); err != nil {
		return err
	}
	var descriptorStr string
	if descriptorStr, err = marshal(descriptor); err != nil {
		return err
	}
	if err := g.StorageClient.WriteToGCS(bucketName, storageutils.ConcatGCSPath(gcsDirectoryPath, instance.Name+".ovf"), strings.NewReader(descriptorStr)); err != nil {
		return err
	}
	return nil
}
