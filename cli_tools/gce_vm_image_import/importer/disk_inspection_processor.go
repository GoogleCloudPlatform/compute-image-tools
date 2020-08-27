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
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type diskInspectionProcessor struct {
	args          ImportArguments
	diskInspector disk.Inspector
}

func (p *diskInspectionProcessor) process(pd persistentDisk) (persistentDisk, error) {
	if !p.args.Inspect || p.diskInspector == nil {
		return pd, nil
	}

	ir, err := p.inspectDisk(pd.uri)
	if err != nil {
		return pd, err
	}

	pd.isUEFICompatible = p.args.UefiCompatible || ir.HasEFIPartition
	pd.isUEFIDetected = ir.HasEFIPartition
	return pd, nil
}

func (p *diskInspectionProcessor) inspectDisk(uri string) (disk.InspectionResult, error) {
	log.Printf("Running disk inspections on %v.", uri)
	ir, err := p.diskInspector.Inspect(uri)
	if err != nil {
		log.Printf("Disk inspection error=%v", err)
		return ir, daisy.Errf("Disk inspection error: %v", err)
	}

	log.Printf("Disk inspection result=%v", ir)
	return ir, nil
}

func (p *diskInspectionProcessor) cancel(reason string) bool {
	if p.diskInspector != nil {
		wf := p.diskInspector.GetWorkflow()
		if wf != nil {
			wf.CancelWithReason(reason)
			return true
		}
	}

	//indicate cancel was not performed
	return false
}

func (p *diskInspectionProcessor) traceLogs() []string {
	if p.diskInspector != nil {
		wf := p.diskInspector.GetWorkflow()
		if wf != nil && wf.Logger != nil {
			return wf.Logger.ReadSerialPortLogs()
		}
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
