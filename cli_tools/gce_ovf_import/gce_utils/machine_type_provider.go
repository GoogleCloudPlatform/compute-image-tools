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
	"sort"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
	daisyCompute "github.com/GoogleCloudPlatform/compute-daisy/compute"
	"github.com/vmware/govmomi/ovf"
	"google.golang.org/api/compute/v1"

	ovfutils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/gce_ovf_import/ovf_utils"
)

// MachineTypeProvider is responsible for providing GCE machine type based virtual appliance
// properties defined in OVF descriptor
type MachineTypeProvider struct {
	OvfDescriptor *ovf.Envelope
	MachineType   string
	ComputeClient daisyCompute.Client
	Project       string
	Zone          string
}

// GetMachineType returns machine type based on properties in OVF descriptor
func (mp *MachineTypeProvider) GetMachineType() (string, error) {
	if mp.MachineType != "" {
		return mp.MachineType, nil
	}

	virtualSystem, err := ovfutils.GetVirtualSystem(mp.OvfDescriptor)
	if err != nil {
		return "", err
	}
	virtualHardwareSection, err := ovfutils.GetVirtualHardwareSection(virtualSystem)
	if err != nil {
		return "", err
	}
	memoryMB, err := ovfutils.GetMemoryInMB(virtualHardwareSection)
	if err != nil {
		return "", err
	}
	cpuCount, err := ovfutils.GetNumberOfCPUs(virtualHardwareSection)
	if err != nil {
		return "", err
	}
	machineTypes, err := mp.ComputeClient.ListMachineTypes(mp.Project, mp.Zone)
	if err != nil {
		return "", err
	}
	cpuCountToMachineTypes, cpuCounts := groupMachineTypesByCPUCount(machineTypes)

	cpuIndex := 0
	memoryIndex := 0

	for cpuIndex < len(cpuCounts) {
		if cpuCounts[cpuIndex] < cpuCount {
			cpuIndex++
			memoryIndex = 0
			continue
		}

		for memoryIndex < len(cpuCountToMachineTypes[cpuCounts[cpuIndex]]) &&
			cpuCountToMachineTypes[cpuCounts[cpuIndex]][memoryIndex].MemoryMb < memoryMB {
			memoryIndex++
		}
		if memoryIndex >= len(cpuCountToMachineTypes[cpuCounts[cpuIndex]]) {
			cpuIndex++
			memoryIndex = 0
			continue
		}
		return cpuCountToMachineTypes[cpuCounts[cpuIndex]][memoryIndex].Name, nil
	}

	return "", daisy.Errf(
		"no machine type has at least %v MBs of memory and %v vCPUs", memoryMB, cpuCount)
}

func groupMachineTypesByCPUCount(
	machineTypes []*compute.MachineType) (map[int64][]*compute.MachineType, []int64) {
	cpuCountToMachineTypes := make(map[int64][]*compute.MachineType)
	cpuCounts := make([]int64, 0)

	for _, machineType := range machineTypes {
		if _, cpuCountExist := cpuCountToMachineTypes[machineType.GuestCpus]; !cpuCountExist {
			cpuCountToMachineTypes[machineType.GuestCpus] = make([]*compute.MachineType, 0)
			cpuCounts = append(cpuCounts, machineType.GuestCpus)
		}
		cpuCountToMachineTypes[machineType.GuestCpus] = append(
			cpuCountToMachineTypes[machineType.GuestCpus], machineType)
	}

	sort.Slice(cpuCounts, func(i, j int) bool { return cpuCounts[i] < cpuCounts[j] })
	for _, cpuCount := range cpuCounts {
		toSort := cpuCountToMachineTypes[cpuCount]
		sort.SliceStable(toSort, func(i, j int) bool {
			return toSort[i].MemoryMb < toSort[j].MemoryMb
		})
	}

	return cpuCountToMachineTypes, cpuCounts
}
