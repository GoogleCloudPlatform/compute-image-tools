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

	commondisk "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
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
type OvfDescriptorGenerator interface {
	GenerateAndWriteOVFDescriptor(instance *compute.Instance, exportedDisks []*ExportedDisk, bucketName, gcsDirectoryPath string, diskInspectionResult *commondisk.InspectionResult) error
	Cancel(reason string) bool
}

type ovfDescriptorGeneratorImpl struct {
	computeClient daisycompute.Client
	storageClient domain.StorageClientInterface
	Project       string
	Zone          string
}

// NewOvfDescriptorGenerator creates a new OvfDescriptorGenerator
func NewOvfDescriptorGenerator(computeClient daisycompute.Client, storageClient domain.StorageClientInterface,
	project string, zone string) OvfDescriptorGenerator {
	return &ovfDescriptorGeneratorImpl{
		computeClient: computeClient,
		storageClient: storageClient,
		Project:       project,
		Zone:          zone,
	}
}

// GenerateAndWriteOVFDescriptor generates an OVF descriptor based on the
// instance exported and disk file paths. and stores it as a file in GCS.
func (g *ovfDescriptorGeneratorImpl) GenerateAndWriteOVFDescriptor(instance *compute.Instance, exportedDisks []*ExportedDisk, bucketName, gcsDirectoryPath string, diskInspectionResult *commondisk.InspectionResult) error {
	var err error
	var descriptor *ovf.Envelope
	if descriptor, err = g.generate(instance, exportedDisks, diskInspectionResult); err != nil {
		return err
	}
	var descriptorStr string
	if descriptorStr, err = marshal(descriptor); err != nil {
		return err
	}
	if err := g.storageClient.WriteToGCS(bucketName, storageutils.ConcatGCSPath(gcsDirectoryPath, instance.Name+".ovf"), strings.NewReader(descriptorStr)); err != nil {
		return err
	}
	return nil
}

// Generate generates an OVF descriptor based on the instance exported and disk file paths.
func (g *ovfDescriptorGeneratorImpl) generate(instance *compute.Instance, exportedDisks []*ExportedDisk, diskInspectionResult *commondisk.InspectionResult) (*ovf.Envelope, error) {
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

	// machine type (CPU, memory)
	if err := g.populateMachineType(instance, descriptor); err != nil {
		return descriptor, err
	}

	// disks
	//TODO: look for AttachedDisk.Boot and export that one first
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

	g.populateOS(descriptor, diskInspectionResult)

	//TODO items: network, video card

	return descriptor, nil
}

func (g *ovfDescriptorGeneratorImpl) populateMachineType(instance *compute.Instance, descriptor *ovf.Envelope) error {
	machineTypeURLSplits := strings.Split(instance.MachineType, "/")
	machineTypeID := machineTypeURLSplits[len(machineTypeURLSplits)-1]
	machineType, err := g.computeClient.GetMachineType(g.Project, g.Zone, machineTypeID)
	if err != nil {
		return err
	}
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item,
		*g.createCPUItem(machineType, strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item)+1)))
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item,
		*g.createMemoryItem(machineType, strconv.Itoa(len(descriptor.VirtualSystem.VirtualHardware[0].Item)+1)))
	return nil
}

func (g *ovfDescriptorGeneratorImpl) populateOS(descriptor *ovf.Envelope, ir *commondisk.InspectionResult) error {
	descriptor.VirtualSystem.OperatingSystem = make([]ovf.OperatingSystemSection, 1)
	descriptor.VirtualSystem.OperatingSystem[0].Info = "The kind of installed guest operating system"

	//default values if no OS deteced
	descriptor.VirtualSystem.OperatingSystem[0].ID = 0
	descriptor.VirtualSystem.OperatingSystem[0].OSType = strPtr("Unknown")
	if ir != nil {
		osInfo, osID := ovfutils.GetOSInfoForInspectionResults(*ir)
		if osInfo != nil {
			descriptor.VirtualSystem.OperatingSystem[0].ID = osID
			//TODO: include minor version as well
			descriptor.VirtualSystem.OperatingSystem[0].Version = strPtr(ir.Major)
			descriptor.VirtualSystem.OperatingSystem[0].OSType = strPtr(osInfo.OsType) //TODO this field is not currently populated in ovf_utils
			descriptor.VirtualSystem.OperatingSystem[0].Description = strPtr(formatOSDescription(ir))
		}
	}
	return nil
}

func formatOSDescription(ir *commondisk.InspectionResult) string {
	//TODO: this can be extracted from virt-inspector output, e.g. <product_name>Windows Server 2012 R2 Standard</product_name>
	desc := strings.Title(fmt.Sprintf("%v %v", ir.Distro, ir.Major))
	if ir.Major != "" {
		desc = fmt.Sprintf("%v.%v", desc, ir.Minor)
	}
	architectureDesc := ""
	switch ir.Architecture {
	case "x86":
		architectureDesc = "32-bit"
	case "x64":
		architectureDesc = "64-bit"
	}
	if architectureDesc != "" {
		desc = fmt.Sprintf("%v (%v)", desc, architectureDesc)
	}
	return desc
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

func (g *ovfDescriptorGeneratorImpl) createCPUItem(machineType *compute.MachineType, instanceID string) *ovf.ResourceAllocationSettingData {
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

func (g *ovfDescriptorGeneratorImpl) createMemoryItem(machineType *compute.MachineType, instanceID string) *ovf.ResourceAllocationSettingData {
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

func (g *ovfDescriptorGeneratorImpl) Cancel(reason string) bool {
	// Descriptor generation is very fast, implementing cancellation is not worth it
	return false
}
