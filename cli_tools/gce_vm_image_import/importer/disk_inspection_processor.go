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

package importer

import (
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/disk"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging/service"
)

// diskInspectionProcessor executes inspection towards the disk, including OS info,
// UEFI partition, etc, so that other processors can consume.
type diskInspectionProcessor struct {
	args          ImportArguments
	diskInspector disk.Inspector
}

func (p *diskInspectionProcessor) process(pd persistentDisk,
	loggableBuilder *service.SingleImageImportLoggableBuilder) (persistentDisk, error) {

	if p.diskInspector == nil {
		return pd, nil
	}

	ir, err := p.inspectDisk(pd.uri)
	if err != nil {
		return pd, err
	}

	isDualBoot := ir.UEFIBootable && ir.BIOSBootableWithHybridMBROrProtectiveMBR
	if !p.args.UefiCompatible && isDualBoot {
		log.Printf("This disk can boot with either BIOS or a UEFI bootloader. The default setting for booting is BIOS. " +
			"If you want to boot using UEFI, please see https://cloud.google.com/compute/docs/import/importing-virtual-disks#importing_a_virtual_disk_with_uefi_bootloader'.")
	}
	pd.isUEFICompatible = p.args.UefiCompatible || (ir.UEFIBootable && !isDualBoot)
	loggableBuilder.SetUEFIMetrics(pd.isUEFICompatible, ir.UEFIBootable, ir.BIOSBootableWithHybridMBROrProtectiveMBR, ir.RootFS)
	return pd, nil
}

func (p *diskInspectionProcessor) inspectDisk(uri string) (disk.InspectionResult, error) {
	log.Printf("Running disk inspections on %v.", uri)
	ir, err := p.diskInspector.Inspect(uri, p.args.Inspect)
	if err != nil {
		log.Printf("Disk inspection error=%v", err)
	}

	log.Printf("Disk inspection result=%v", ir)
	return ir, nil
}

func (p *diskInspectionProcessor) cancel(reason string) bool {
	if p.diskInspector != nil {
		return p.diskInspector.Cancel(reason)
	}

	//indicate cancel was not performed
	return false
}

func (p *diskInspectionProcessor) traceLogs() []string {
	if p.diskInspector != nil {
		return p.diskInspector.TraceLogs()
	}
	return []string{}
}

func newDiskInspectionProcessor(diskInspector disk.Inspector,
	args ImportArguments) processor {

	return &diskInspectionProcessor{
		args:          args,
		diskInspector: diskInspector,
	}
}
