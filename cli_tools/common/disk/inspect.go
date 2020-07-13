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

package disk

import (
	"context"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const workflowFile = "daisy_workflows/image_import/inspection/inspect-disk.wf.json"

// Inspector finds partition and boot-related properties for a disk.
type Inspector interface {
	Inspect(reference string) (Disk, error)
}

// NewInspector creates an Inspector that can inspect GCP disks.
func NewInspector(network, subnet string, wfAttributes daisycommon.WorkflowAttributes) (Inspector, error) {
	wf, err := daisy.NewFromFile(workflowFile)
	if err != nil {
		return nil, err
	}
	wf.AddVar("network", network)
	wf.AddVar("subnet", subnet)
	daisycommon.SetWorkflowAttributes(wf, wfAttributes)
	return defaultInspector{wf}, nil
}

// defaultInspector implements disk.Inspector using a Daisy workflow.
type defaultInspector struct {
	wf *daisy.Workflow
}

func (inspector defaultInspector) Inspect(reference string) (Disk, error) {
	inspector.wf.AddVar("pd_uri", reference)
	err := inspector.wf.Run(context.Background())
	if err != nil {
		return Disk{}, err
	}
	return populateDisk(inspector.wf), err
}

func populateDisk(wf *daisy.Workflow) Disk {
	scheme := PartitionScheme_NONE
	switch wf.GetSerialConsoleOutputValue("partition_scheme") {
	case "mbr":
		scheme = PartitionScheme_MBR
	case "gpt":
		scheme = PartitionScheme_GPT
	}
	return Disk{
		PartitionScheme: scheme,
	}
}
