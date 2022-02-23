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
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"google.golang.org/api/compute/v1"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/domain"
	storageutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/storage"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_export/exporter/ovf"
	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
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

type ovfDescriptorGeneratorImpl struct {
	computeClient daisyCompute.Client
	storageClient domain.StorageClientInterface
	Project       string
	Zone          string
}

// NewOvfDescriptorGenerator creates a new OvfDescriptorGenerator
func NewOvfDescriptorGenerator(computeClient daisyCompute.Client, storageClient domain.StorageClientInterface,
	project string, zone string) ovfexportdomain.OvfDescriptorGenerator {
	return &ovfDescriptorGeneratorImpl{
		computeClient: computeClient,
		storageClient: storageClient,
		Project:       project,
		Zone:          zone,
	}
}

// GenerateAndWriteOVFDescriptor generates an OVF descriptor based on the
// instance exported and disk file paths. and stores it as a file in GCS.
func (g *ovfDescriptorGeneratorImpl) GenerateAndWriteOVFDescriptor(
	instance *compute.Instance, exportedDisks []*ovfexportdomain.ExportedDisk,
	bucketName, gcsDirectoryPath, descriptorFileName string,
	diskInspectionResult *pb.InspectionResults) error {

	var err error
	var descriptor *ovf.Envelope
	if descriptor, err = g.generate(instance, exportedDisks, diskInspectionResult); err != nil {
		return err
	}
	var descriptorStr string
	if descriptorStr, err = marshal(descriptor); err != nil {
		return err
	}
	if err := g.storageClient.WriteToGCS(
		bucketName,
		storageutils.ConcatGCSPath(gcsDirectoryPath, descriptorFileName),
		strings.NewReader(descriptorStr)); err != nil {
		return err
	}
	return nil
}

// Generate generates an OVF descriptor based on the instance exported and disk file paths.
func (g *ovfDescriptorGeneratorImpl) generate(instance *compute.Instance, exportedDisks []*ovfexportdomain.ExportedDisk, diskInspectionResult *pb.InspectionResults) (*ovf.Envelope, error) {
	descriptor := &ovf.Envelope{}
	descriptor.References = make([]ovf.File, len(exportedDisks))
	descriptor.Disk = &ovf.DiskSection{Disks: make([]ovf.VirtualDiskDesc, len(exportedDisks)), Section: ovf.Section{Info: "Virtual disk information"}}

	descriptor.VirtualSystem = &ovf.VirtualSystem{Content: ovf.Content{Info: "A GCE virtual machine", ID: instance.Name, Name: &instance.Name}}
	descriptor.VirtualSystem.VirtualHardware = make([]ovf.VirtualHardwareSection, 1)
	descriptor.VirtualSystem.VirtualHardware[0] = ovf.VirtualHardwareSection{Section: ovf.Section{Info: "Virtual hardware requirements"}}
	descriptor.VirtualSystem.VirtualHardware[0].System = &ovf.VirtualSystemSettingData{CIMVirtualSystemSettingData: ovf.CIMVirtualSystemSettingData{ElementName: "Virtual Hardware Family", InstanceID: "0", VirtualSystemIdentifier: &instance.Name}}

	// disk controller
	discControllerName := "SCSI Controller"
	diskController := ovf.ResourceAllocationSettingData{
		CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
			ElementName:  discControllerName,
			Description:  strPtr(discControllerName),
			InstanceID:   generateVirtualHardwareItemID(descriptor),
			ResourceType: func() *uint16 { v := iSCSIController; return &v }(),
		},
	}
	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item, diskController)

	// machine type (CPU, memory)
	if err := g.populateMachineType(instance, descriptor); err != nil {
		return descriptor, err
	}

	// disks
	// first add boot disk...
	for diskIndex, exportedDisk := range exportedDisks {
		if exportedDisk.AttachedDisk.Boot {
			g.addDisk(exportedDisk, descriptor, diskIndex, diskController.InstanceID)
		}
	}
	//...then data disks
	for diskIndex, exportedDisk := range exportedDisks {
		if !exportedDisk.AttachedDisk.Boot {
			g.addDisk(exportedDisk, descriptor, diskIndex, diskController.InstanceID)
		}
	}

	g.populateOS(descriptor, diskInspectionResult)

	return descriptor, nil
}

func (g *ovfDescriptorGeneratorImpl) addDisk(exportedDisk *ovfexportdomain.ExportedDisk, descriptor *ovf.Envelope, diskIndex int, diskControllerInstanceID string) {
	diskGCSFileName := exportedDisk.GcsPath
	if slash := strings.LastIndex(diskGCSFileName, "/"); slash > -1 {
		diskGCSFileName = diskGCSFileName[slash+1:]
	}
	descriptor.References[diskIndex] = ovf.File{
		Href: diskGCSFileName,
		ID:   "file" + strconv.Itoa(diskIndex),
		Size: uint(exportedDisk.GcsFileAttrs.Size),
	}

	descriptor.Disk.Disks[diskIndex] = ovf.VirtualDiskDesc{
		DiskID:   fmt.Sprintf("vmdisk%v", diskIndex),
		FileRef:  &descriptor.References[diskIndex].ID,
		Capacity: strconv.FormatInt(exportedDisk.Disk.SizeGb*bytesPerGB, 10),
	}

	descriptor.VirtualSystem.VirtualHardware[0].Item = append(
		descriptor.VirtualSystem.VirtualHardware[0].Item,
		*createDiskItem(generateVirtualHardwareItemID(descriptor),
			diskControllerInstanceID,
			diskIndex,
			descriptor.Disk.Disks[diskIndex].DiskID,
		),
	)
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

func (g *ovfDescriptorGeneratorImpl) populateOS(descriptor *ovf.Envelope, ir *pb.InspectionResults) error {
	descriptor.VirtualSystem.OperatingSystem = make([]ovf.OperatingSystemSection, 1)
	descriptor.VirtualSystem.OperatingSystem[0].Info = "The kind of installed guest operating system"

	//default values if no OS deteced
	descriptor.VirtualSystem.OperatingSystem[0].ID = 0
	descriptor.VirtualSystem.OperatingSystem[0].OSType = strPtr("Unknown")
	if ir != nil {
		osInfo, osID := ovfutils.GetOSInfoForInspectionResults(ir)
		if osInfo != nil {
			descriptor.VirtualSystem.OperatingSystem[0].ID = osID
			//TODO: include minor version as well
			descriptor.VirtualSystem.OperatingSystem[0].Version = strPtr(ir.GetOsRelease().GetMajorVersion())
			descriptor.VirtualSystem.OperatingSystem[0].OSType = strPtr(osInfo.OsType) //TODO this field is not currently populated in ovf_utils
			descriptor.VirtualSystem.OperatingSystem[0].Description = strPtr(formatOSDescription(ir))
		}
	}
	return nil
}

func formatOSDescription(ir *pb.InspectionResults) string {
	//TODO: this can be extracted from virt-inspector output, e.g. <product_name>Windows Server 2012 R2 Standard</product_name
	desc := strings.Title(fmt.Sprintf("%v %v", ir.GetOsRelease().GetDistro(), ir.GetOsRelease().GetMajorVersion()))
	if ir.GetOsRelease().GetMinorVersion() != "" {
		desc = fmt.Sprintf("%v.%v", desc, ir.GetOsRelease().GetMinorVersion())
	}
	architectureDesc := ""
	switch ir.GetOsRelease().GetArchitecture() {
	case pb.Architecture_X86:
		architectureDesc = "32-bit"
	case pb.Architecture_X64:
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
			VirtualQuantity: func() *uint { v := uint(machineType.MemoryMb); return &v }(),
		},
	}
}

func marshal(descriptor *ovf.Envelope) (string, error) {
	descriptor.XMLNSCIM = "http://schemas.dmtf.org/wbem/wscim/1/common"
	descriptor.XMLNSOVF = "http://schemas.dmtf.org/ovf/envelope/1"
	descriptor.XMLNSRASD = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_ResourceAllocationSettingData"
	descriptor.XMLNSVMW = "http://www.vmware.com/schema/ovf"
	descriptor.XMLNSVSSD = "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_VirtualSystemSettingData"
	descriptor.XMLNSXSI = "http://www.w3.org/2001/XMLSchema-instance"

	raw, err := xml.MarshalIndent(&descriptor, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func strPtr(s string) *string {
	return &s
}

func (g *ovfDescriptorGeneratorImpl) Cancel(reason string) bool {
	// Descriptor generation is very fast, implementing cancellation is not worth it
	return false
}
