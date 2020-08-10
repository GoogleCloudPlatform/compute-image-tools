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

// OvfDescriptorGenerator is responsible for generating OVF descriptor based on
//GCE instance being exported.
type OvfDescriptorGenerator struct {
	ComputeClient daisycompute.Client
	StorageClient domain.StorageClientInterface
	Project       string
	Zone          string
}

// Generate generates an OVF descriptor based on the instance exported and disk file paths.
func (g *OvfDescriptorGenerator) Generate(instance *compute.Instance, exportedDisks []*ExportedDisk) (*ovf.Envelope, error) {
	descriptor := &ovf.Envelope{}
	descriptor.VirtualHardware = &ovf.VirtualHardwareSection{}
	descriptor.References = make([]ovf.File, len(exportedDisks))
	descriptor.Disk = &ovf.DiskSection{Disks: make([]ovf.VirtualDiskDesc, len(exportedDisks)), Section: ovf.Section{Info: "Virtual disk information"}}

	descriptor.VirtualSystem = &ovf.VirtualSystem{Content: ovf.Content{Info: "A GCE virtual machine", Name: &instance.Name}}
	descriptor.VirtualSystem.VirtualHardware = make([]ovf.VirtualHardwareSection, 1)
	descriptor.VirtualSystem.VirtualHardware[0] = ovf.VirtualHardwareSection{Section: ovf.Section{Info: "Virtual hardware requirements"}}

	//TODO: do we need VirtualSystemType? It's set to e.g. "vmx-11". It's a VMWare identifier
	descriptor.VirtualSystem.VirtualHardware[0].System = &ovf.VirtualSystemSettingData{CIMVirtualSystemSettingData: ovf.CIMVirtualSystemSettingData{ElementName: "Virtual Hardware Family", InstanceID: "0", VirtualSystemIdentifier: &instance.Name}}

	// disk controller
	disController := ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			Description:  strPtr("SCSI Controller"),
			InstanceID:   generateVirtualHardwareItemID(descriptor),
			ResourceType: func() *uint16 { v := iSCSIController; return &v }(),
		},
	}
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item, disController)

	// disks
	for diskIndex, exportedDisk := range exportedDisks {

		diskGCSFileName := exportedDisk.gcsPath
		if slash := strings.LastIndex(diskGCSFileName, "/"); slash > -1 {
			diskGCSFileName = diskGCSFileName[slash+1:]
		}
		descriptor.References[diskIndex] = ovf.File{
			Href: diskGCSFileName,
			ID:   "file" + strconv.Itoa(diskIndex),
			Size: uint(exportedDisk.gcsFileAttrs.Size),
		}

		descriptor.Disk.Disks[diskIndex] = ovf.VirtualDiskDesc{
			DiskID:                  fmt.Sprintf("vmdisk%v", diskIndex),
			FileRef:                 &descriptor.References[diskIndex].ID,
			Capacity:                strconv.FormatInt(exportedDisk.disk.SizeGb, 10),
			CapacityAllocationUnits: strPtr("byte * 2^30"),
		}

		//TODO: virtual hardware section for ordering? might not be necessary for export. We used it for import to determine order of disks if disks are spread over multiple controllers
		descriptor.VirtualSystem.VirtualHardware[0].Item = append(
			descriptor.VirtualSystem.VirtualHardware[0].Item,
			*createDiskItem(generateVirtualHardwareItemID(descriptor), disController.InstanceID, diskIndex, descriptor.Disk.Disks[diskIndex].DiskID))
	}

	//TODO: operating system
	descriptor.VirtualSystem.OperatingSystem = make([]ovf.OperatingSystemSection, 1)
	descriptor.VirtualSystem.OperatingSystem[0].ID = 0
	descriptor.VirtualSystem.OperatingSystem[0].OSType = strPtr("Unknown")
	descriptor.VirtualSystem.OperatingSystem[0].Info = "The kind of installed guest operating system"

	//TODO items: network, video card

	// machine type (CPU, memory)
	machineTypeURLSplits := strings.Split(instance.MachineType, "/")
	machineTypeID := machineTypeURLSplits[len(machineTypeURLSplits)-1]
	machineType, err := g.ComputeClient.GetMachineType(g.Project, g.Zone, machineTypeID)
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

func generateVirtualHardwareItemID(descriptor *ovf.Envelope) string {
	return strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item) + 1)
}

func appendItem(item *ovf.ResourceAllocationSettingData, items *[]ovf.ResourceAllocationSettingData) {
	*items = append(*items, *item)
}

func createDiskItem(instanceID string, parentID string, addressOnParent int, diskID string) *ovf.ResourceAllocationSettingData {
	return &ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			AddressOnParent: strPtr(strconv.Itoa(addressOnParent)),
			ElementName:     fmt.Sprintf("disk%v", addressOnParent),
			HostResource:    []string{fmt.Sprintf("ovf:/disk/%v", diskID)},
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

// GenerateAndWriteOVFDescriptor generates an OVF descriptor based on the
// instance exported and disk file paths. and stores it as a file in GCS.
func (g *OvfDescriptorGenerator) GenerateAndWriteOVFDescriptor(instance *compute.Instance, exportedDisks []*ExportedDisk, bucketName, gcsDirectoryPath string) error {
	var err error
	var descriptor *ovf.Envelope
	if descriptor, err = g.Generate(instance, exportedDisks); err != nil {
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
