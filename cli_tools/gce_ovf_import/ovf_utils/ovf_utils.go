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
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/domain"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
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

type osInfo struct {
	// description holds OS description that can be used for messages shown to users
	description string

	// importerOSIDs holds a list of OS IDs as expected by the importer (--os flag)
	importerOSIDs []string

	// nonDeterministic when set to true indicates OVF import can't determine
	// which OS to use for translate because the mapping is not 1:1 or we don't
	// want to assume the mapping due to potential legal issues. For example,
	// windows 7/8/10 (client) images can only be imported as BYOL but we don't
	// know if BYOL agreement is in place for the customer. In that case, customers
	// have to explicitly provide --os flag.
	nonDeterministic bool
}

// isDeterministic returns true if, based on osInfo, importer OS ID can be determined.
// if false is returned, the customer needs to provide OS value via the --os flag.
func (oi *osInfo) isDeterministic() bool {
	return len(oi.importerOSIDs) == 1 && !oi.nonDeterministic
}

// hasImporterOSIDs returns true if osInfo has any mappings to importer OS IDs.
func (oi *osInfo) hasImporterOSIDs() bool {
	return len(oi.importerOSIDs) > 0
}

//Mapping OVF OS ID to OS info
//Full list: http://schemas.dmtf.org/wbem/cim-html/2/CIM_OperatingSystem.html
var ovfOSIDToImporterOSID = map[int16]osInfo{
	2:   {description: "MACOS", importerOSIDs: []string{}},
	3:   {description: "ATTUNIX", importerOSIDs: []string{}},
	4:   {description: "DGUX", importerOSIDs: []string{}},
	5:   {description: "DECNT", importerOSIDs: []string{}},
	6:   {description: "Tru64 UNIX", importerOSIDs: []string{}},
	7:   {description: "OpenVMS", importerOSIDs: []string{}},
	8:   {description: "HPUX", importerOSIDs: []string{}},
	9:   {description: "AIX", importerOSIDs: []string{}},
	10:  {description: "MVS", importerOSIDs: []string{}},
	11:  {description: "OS400", importerOSIDs: []string{}},
	12:  {description: "OS/2", importerOSIDs: []string{}},
	13:  {description: "JavaVM", importerOSIDs: []string{}},
	14:  {description: "MSDOS", importerOSIDs: []string{}},
	15:  {description: "WIN3x", importerOSIDs: []string{}},
	16:  {description: "WIN95", importerOSIDs: []string{}},
	17:  {description: "WIN98", importerOSIDs: []string{}},
	18:  {description: "WINNT", importerOSIDs: []string{}},
	19:  {description: "WINCE", importerOSIDs: []string{}},
	20:  {description: "NCR3000", importerOSIDs: []string{}},
	21:  {description: "NetWare", importerOSIDs: []string{}},
	22:  {description: "OSF", importerOSIDs: []string{}},
	23:  {description: "DC/OS", importerOSIDs: []string{}},
	24:  {description: "Reliant UNIX", importerOSIDs: []string{}},
	25:  {description: "SCO UnixWare", importerOSIDs: []string{}},
	26:  {description: "SCO OpenServer", importerOSIDs: []string{}},
	27:  {description: "Sequent", importerOSIDs: []string{}},
	28:  {description: "IRIX", importerOSIDs: []string{}},
	29:  {description: "Solaris", importerOSIDs: []string{}},
	30:  {description: "SunOS", importerOSIDs: []string{}},
	31:  {description: "U6000", importerOSIDs: []string{}},
	32:  {description: "ASERIES", importerOSIDs: []string{}},
	33:  {description: "HP NonStop OS", importerOSIDs: []string{}},
	34:  {description: "HP NonStop OSS", importerOSIDs: []string{}},
	35:  {description: "BS2000", importerOSIDs: []string{}},
	36:  {description: "LINUX", importerOSIDs: []string{}},
	37:  {description: "Lynx", importerOSIDs: []string{}},
	38:  {description: "XENIX", importerOSIDs: []string{}},
	39:  {description: "VM", importerOSIDs: []string{}},
	40:  {description: "Interactive UNIX", importerOSIDs: []string{}},
	41:  {description: "BSDUNIX", importerOSIDs: []string{}},
	42:  {description: "FreeBSD", importerOSIDs: []string{}},
	43:  {description: "NetBSD", importerOSIDs: []string{}},
	44:  {description: "GNU Hurd", importerOSIDs: []string{}},
	45:  {description: "OS9", importerOSIDs: []string{}},
	46:  {description: "MACH Kernel", importerOSIDs: []string{}},
	47:  {description: "Inferno", importerOSIDs: []string{}},
	48:  {description: "QNX", importerOSIDs: []string{}},
	49:  {description: "EPOC", importerOSIDs: []string{}},
	50:  {description: "IxWorks", importerOSIDs: []string{}},
	51:  {description: "VxWorks", importerOSIDs: []string{}},
	52:  {description: "MiNT", importerOSIDs: []string{}},
	53:  {description: "BeOS", importerOSIDs: []string{}},
	54:  {description: "HP MPE", importerOSIDs: []string{}},
	55:  {description: "NextStep", importerOSIDs: []string{}},
	56:  {description: "PalmPilot", importerOSIDs: []string{}},
	57:  {description: "Rhapsody", importerOSIDs: []string{}},
	58:  {description: "Windows 2000", importerOSIDs: []string{}},
	59:  {description: "Dedicated", importerOSIDs: []string{}},
	60:  {description: "OS/390", importerOSIDs: []string{}},
	61:  {description: "VSE", importerOSIDs: []string{}},
	62:  {description: "TPF", importerOSIDs: []string{}},
	63:  {description: "Windows (R) Me", importerOSIDs: []string{}},
	64:  {description: "Caldera Open UNIX", importerOSIDs: []string{}},
	65:  {description: "OpenBSD", importerOSIDs: []string{}},
	66:  {description: "Not Applicable", importerOSIDs: []string{}},
	67:  {description: "Windows XP", importerOSIDs: []string{}},
	68:  {description: "z/OS", importerOSIDs: []string{}},
	69:  {description: "Microsoft Windows Server 2003", importerOSIDs: []string{}},
	70:  {description: "Microsoft Windows Server 2003 64-Bit", importerOSIDs: []string{}},
	71:  {description: "Windows XP 64-Bit", importerOSIDs: []string{}},
	72:  {description: "Windows XP Embedded", importerOSIDs: []string{}},
	73:  {description: "Windows Vista", importerOSIDs: []string{}},
	74:  {description: "Windows Vista 64-Bit", importerOSIDs: []string{}},
	75:  {description: "Windows Embedded for Point of Service", importerOSIDs: []string{}},
	76:  {description: "Microsoft Windows Server 2008", importerOSIDs: []string{}},
	77:  {description: "Microsoft Windows Server 2008 64-Bit", importerOSIDs: []string{}},
	78:  {description: "FreeBSD 64-Bit", importerOSIDs: []string{}},
	79:  {description: "RedHat Enterprise Linux", importerOSIDs: []string{}},
	80:  {description: "RedHat Enterprise Linux 64-Bit", importerOSIDs: []string{"rhel-6", "rhel-6-byol", "rhel-7", "rhel-7-byol", "rhel-8", "rhel-8-byol"}},
	81:  {description: "Solaris 64-Bit", importerOSIDs: []string{}},
	82:  {description: "SUSE", importerOSIDs: []string{}},
	83:  {description: "SUSE 64-Bit", importerOSIDs: []string{"opensuse-15"}, nonDeterministic: true},
	84:  {description: "SLES", importerOSIDs: []string{}},
	85:  {description: "SLES 64-Bit", importerOSIDs: []string{"sles-12-byol", "sles-15-byol"}},
	86:  {description: "Novell OES", importerOSIDs: []string{}},
	87:  {description: "Novell Linux Desktop", importerOSIDs: []string{}},
	88:  {description: "Sun Java Desktop System", importerOSIDs: []string{}},
	89:  {description: "Mandriva", importerOSIDs: []string{}},
	90:  {description: "Mandriva 64-Bit", importerOSIDs: []string{}},
	91:  {description: "TurboLinux", importerOSIDs: []string{}},
	92:  {description: "TurboLinux 64-Bit", importerOSIDs: []string{}},
	93:  {description: "Ubuntu", importerOSIDs: []string{}},
	94:  {description: "Ubuntu 64-Bit", importerOSIDs: []string{"ubuntu-1404", "ubuntu-1604", "ubuntu-1804"}},
	95:  {description: "Debian", importerOSIDs: []string{}},
	96:  {description: "Debian 64-Bit", importerOSIDs: []string{"debian-8", "debian-9"}},
	97:  {description: "Linux 2.4.x", importerOSIDs: []string{}},
	98:  {description: "Linux 2.4.x 64-Bit", importerOSIDs: []string{}},
	99:  {description: "Linux 2.6.x", importerOSIDs: []string{}},
	100: {description: "Linux 2.6.x 64-Bit", importerOSIDs: []string{}},
	101: {description: "Linux 64-Bit", importerOSIDs: []string{}},
	103: {description: "Microsoft Windows Server 2008 R2", importerOSIDs: []string{"windows-2008r2", "windows-2008r2-byol"}},
	104: {description: "VMware ESXi", importerOSIDs: []string{}},
	105: {description: "Microsoft Windows 7", importerOSIDs: []string{"windows-7-x64-byol", "windows-7-x86-byol"}},
	106: {description: "CentOS 32-bit", importerOSIDs: []string{}},
	107: {description: "CentOS 64-bit", importerOSIDs: []string{"centos-6", "centos-7", "centos-8"}},
	108: {description: "Oracle Linux 32-bit", importerOSIDs: []string{}},
	109: {description: "Oracle Linux 64-bit", importerOSIDs: []string{}},
	110: {description: "eComStation 32-bitx", importerOSIDs: []string{}},
	111: {description: "Microsoft Windows Server 2011", importerOSIDs: []string{}},
	113: {description: "Microsoft Windows Server 2012", importerOSIDs: []string{"windows-2012", "windows-2012-byol"}},
	114: {description: "Microsoft Windows 8", importerOSIDs: []string{"windows-8-x86-byol"}, nonDeterministic: true},
	115: {description: "Microsoft Windows 8 64-bit", importerOSIDs: []string{"windows-8-x64-byol"}, nonDeterministic: true},
	116: {description: "Microsoft Windows Server 2012 R2", importerOSIDs: []string{"windows-2012r2", "windows-2012r2-byol"}},
	117: {description: "Microsoft Windows Server 2016", importerOSIDs: []string{"windows-2016", "windows-2016-byol"}},
	118: {description: "Microsoft Windows 8.1", importerOSIDs: []string{}},
	119: {description: "Microsoft Windows 8.1 64-bit", importerOSIDs: []string{}},
	120: {description: "Microsoft Windows 10", importerOSIDs: []string{"windows-10-x86-byol"}, nonDeterministic: true},
	121: {description: "Microsoft Windows 10 64-bit", importerOSIDs: []string{"windows-10-x64-byol"}, nonDeterministic: true},
	//TODO: windows-2019, windows-2019-byol
}

//Mapping OVF osType attribute to importer OS ID
//Some might have multiple importer OS ID  values. In that case, user need to provide a value via --os flag.
// Some might have only one option but we can't select it automatically as we cannot guarantee
// correctness. All Windows Client imports are in this category due to the fact we can't assume BYOL licensing.
//Full list: https://vdc-download.vmware.com/vmwb-repository/dcr-public/da47f910-60ac-438b-8b9b-6122f4d14524/16b7274a-bf8b-4b4c-a05e-746f2aa93c8c/doc/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html
var ovfOSTypeToOSID = map[string]osInfo{
	"debian8_64Guest":       osInfo{importerOSIDs: []string{"debian-8"}},
	"debian9_64Guest":       osInfo{importerOSIDs: []string{"debian-9"}},
	"centos6_64Guest":       osInfo{importerOSIDs: []string{"centos-6"}},
	"centos7_64Guest":       osInfo{importerOSIDs: []string{"centos-7"}},
	"centos8_64Guest":       osInfo{importerOSIDs: []string{"centos-8"}},
	"rhel6_64Guest":         osInfo{importerOSIDs: []string{"rhel-6"}},
	"rhel7_64Guest":         osInfo{importerOSIDs: []string{"rhel-7"}},
	"windows7Server64Guest": osInfo{importerOSIDs: []string{"windows-2008r2"}},
	"ubuntu64Guest":         {importerOSIDs: []string{"ubuntu-1404", "ubuntu-1604", "ubuntu-1804"}, nonDeterministic: true},
	"windows7Guest":         {importerOSIDs: []string{"windows-7-x86-byol"}, nonDeterministic: true},
	"windows7_64Guest":      {importerOSIDs: []string{"windows-7-x64-byol"}, nonDeterministic: true},
	"windows8Guest":         {importerOSIDs: []string{"windows-8-x86-byol"}, nonDeterministic: true},
	"windows8_64Guest":      {importerOSIDs: []string{"windows-8-x64-byol"}, nonDeterministic: true},
	"windows9Guest":         {importerOSIDs: []string{"windows-10-x86-byol"}, nonDeterministic: true},
	"windows9_64Guest":      {importerOSIDs: []string{"windows-10-x64-byol"}, nonDeterministic: true},
	"windows8Server64Guest": {importerOSIDs: []string{"windows-2012", "windows-2012r2", "windows-2012-byol", "windows-2012r2-byol"}},
	"windows9Server64Guest": {importerOSIDs: []string{"windows-2016", "windows-2016-byol", "windows-2019", "windows-2019-byol"}},
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
		return nil, daisy.Errf("virtualHardware cannot be nil")
	}
	if diskSection == nil || diskSection.Disks == nil || len(diskSection.Disks) == 0 {
		return nil, daisy.Errf("diskSection cannot be nil")
	}
	if references == nil || *references == nil {
		return nil, daisy.Errf("references cannot be nil")
	}

	diskControllers := getDiskControllersPrioritized(virtualHardware)
	if len(diskControllers) == 0 {
		return nil, daisy.Errf("no disk controllers found in OVF, can't retrieve disk info")
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
		return 0, daisy.Errf("virtualHardware cannot be nil")
	}

	cpuItems := filterItemsByResourceTypes(virtualHardware, cpu)
	if len(cpuItems) == 0 {
		return 0, daisy.Errf("no CPUs found in OVF")
	}

	// Returning the first CPU item found. Doesn't support multiple deployment configurations.
	return int64(*cpuItems[0].VirtualQuantity), nil
}

// GetMemoryInMB returns memory size in MB from OVF virtualHardware section. If there are multiple
// elements defining memory for the same virtual system, the first memory element will be used.
func GetMemoryInMB(virtualHardware *ovf.VirtualHardwareSection) (int64, error) {
	if virtualHardware == nil {
		return 0, daisy.Errf("virtualHardware cannot be nil")
	}

	memoryItems := filterItemsByResourceTypes(virtualHardware, memory)
	if len(memoryItems) == 0 {
		return 0, daisy.Errf("no memory section found in OVF")
	}

	// Using the first memory item found. Doesn't support multiple deployment configurations.
	memoryItem := memoryItems[0]
	if memoryItem.AllocationUnits == nil || *memoryItem.AllocationUnits == "" {
		return 0, daisy.Errf("memory allocation unit not specified")
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
		return nil, daisy.Errf("virtual system is nil, can't extract Virtual hardware")
	}
	if virtualSystem.VirtualHardware == nil || len(virtualSystem.VirtualHardware) == 0 {
		return nil, daisy.Errf("virtual hardware is nil or empty")
	}
	return &virtualSystem.VirtualHardware[0], nil
}

// GetVirtualSystem returns VirtualSystem element from OVF descriptor envelope
func GetVirtualSystem(ovfDescriptor *ovf.Envelope) (*ovf.VirtualSystem, error) {
	if ovfDescriptor == nil {
		return nil, daisy.Errf("OVF descriptor is nil, can't extract virtual system")
	}
	if ovfDescriptor.VirtualSystem == nil {
		return nil, daisy.Errf("OVF descriptor doesn't contain a virtual system")
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
		return "", daisy.Errf("OVF descriptor error: VirtualSystem must be defined to retrieve OS info. Use --os flag to specify OS")
	}
	if ovfDescriptor.VirtualSystem.OperatingSystem == nil ||
		len(ovfDescriptor.VirtualSystem.OperatingSystem) == 0 {
		return "", daisy.Errf("OVF descriptor error: OperatingSystemSection must be defined to retrieve OS info. Use --os flag to specify OS")
	}
	if ovfDescriptor.VirtualSystem.OperatingSystem[0].OSType == nil && ovfDescriptor.VirtualSystem.OperatingSystem[0].ID == 0 {
		return "", daisy.Errf("OVF descriptor error: OperatingSystemSection.OSType or OperatingSystemSection.ID must be defined to retrieve OS info. Use --os flag to specify OS")
	}

	var osInfoFromOSType, osInfoFromOSID *osInfo
	if ovfDescriptor.VirtualSystem.OperatingSystem[0].OSType != nil && *ovfDescriptor.VirtualSystem.OperatingSystem[0].OSType != "" {
		if osInfoFromOSTypeValue, ok := ovfOSTypeToOSID[*ovfDescriptor.VirtualSystem.OperatingSystem[0].OSType]; ok {
			osInfoFromOSType = &osInfoFromOSTypeValue
		}
	}

	if ovfDescriptor.VirtualSystem.OperatingSystem[0].ID != 0 {
		if osInfoFromOSIDValue, ok := ovfOSIDToImporterOSID[ovfDescriptor.VirtualSystem.OperatingSystem[0].ID]; ok {
			osInfoFromOSID = &osInfoFromOSIDValue
		}
	}

	if osInfoFromOSType == nil && osInfoFromOSID == nil {
		return "", daisy.Errf("cannot determine OS from OVF descriptor. Use --os flag to specify OS")
	}

	osInfo := getMoreSpecificOSInfo(osInfoFromOSType, osInfoFromOSID)

	if !osInfo.hasImporterOSIDs() {
		return "", daisy.Errf("operating system `%v` detected but is not supported by Google Compute Engine. Use --os flag to specify OS", osInfo.description)
	}
	if osInfo.isDeterministic() {
		return osInfo.importerOSIDs[0], nil
	}

	return "",
		daisy.Errf(
			"cannot determine OS from OVF descriptor. Use --os flag to specify OS. Potential valid values for given osType attribute are: %v",
			strings.Join(osInfo.importerOSIDs, ", "),
		)
}

func getMoreSpecificOSInfo(osInfo1, osInfo2 *osInfo) *osInfo {
	if osInfo1 == nil || !osInfo1.hasImporterOSIDs() {
		return osInfo2
	}
	if osInfo2 == nil || !osInfo2.hasImporterOSIDs() {
		return osInfo1
	}
	if osInfo1.isDeterministic() {
		return osInfo1
	}
	if osInfo2.isDeterministic() {
		return osInfo2
	}
	if len(osInfo1.importerOSIDs) < len(osInfo2.importerOSIDs) {
		return osInfo1
	}
	return osInfo2
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
	allItems := make([]ovf.ResourceAllocationSettingData, len(virtualHardware.Item))
	copy(allItems, virtualHardware.Item)

	for _, storageItem := range virtualHardware.StorageItem {
		allItems = append(allItems, ovf.ResourceAllocationSettingData{
			CIMResourceAllocationSettingData: ovf.CIMResourceAllocationSettingData{
				AddressOnParent: storageItem.AddressOnParent,
				Caption:         storageItem.Caption,
				Description:     storageItem.Description,
				HostResource:    storageItem.HostResource,
				InstanceID:      storageItem.InstanceID,
				Parent:          storageItem.Parent,
				ResourceType:    storageItem.ResourceType,
			},
		})
	}

	filteredItems := make([]ovf.ResourceAllocationSettingData, 0)
	for _, item := range allItems {
		for _, resourceType := range resourceTypes {
			if *item.ResourceType == resourceType {
				filteredItems = append(filteredItems, item)
			}
		}
	}
	return filteredItems
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
			return "", nil, daisy.Errf("file reference '%v' for disk '%v' not found in OVF descriptor", *disk.FileRef, diskID)
		}
	}
	return "", nil, daisy.Errf(
		"disk with reference %v couldn't be found in OVF descriptor", diskHostResource)
}

func extractDiskID(diskHostResource string) (string, error) {
	if strings.HasPrefix(diskHostResource, "ovf:/disk/") {
		return strings.TrimPrefix(diskHostResource, "ovf:/disk/"), nil
	} else if strings.HasPrefix(diskHostResource, "/disk/") {
		return strings.TrimPrefix(diskHostResource, "/disk/"), nil
	}

	return "", daisy.Errf("disk host resource %v has invalid format", diskHostResource)
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
