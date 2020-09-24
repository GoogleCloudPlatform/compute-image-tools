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
	"path"
	"strconv"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	workflowFile = "image_import/inspection/boot-inspect.wf.json"
)

// Inspector finds partition and boot-related properties for a disk.
type Inspector interface {
	// Inspect finds partition and boot-related properties for a disk and
	// returns an InspectionResult. The reference is implementation specific.
	Inspect(reference string, inspectOS bool) (InspectionResult, error)
	Cancel(reason string) bool
	TraceLogs() []string
}

// InspectionResult contains the partition and boot-related properties of a disk.
type InspectionResult struct {
	// UEFIBootable indicates whether the disk has a UEFI boot loader.
	UEFIBootable bool

	// BIOSBootable indicates whether the disk has a BIOS boot loader.
	BIOSBootable bool

	// RootFS indicates the file system type of the partition containing
	// the root directory ("/").
	RootFS string

	Architecture, Distro, Major, Minor string
}

// NewInspector creates an Inspector that can inspect GCP disks.
func NewInspector(wfAttributes daisycommon.WorkflowAttributes) (Inspector, error) {
	wf, err := daisy.NewFromFile(path.Join(wfAttributes.WorkflowDirectory, workflowFile))
	if err != nil {
		return nil, err
	}
	daisycommon.SetWorkflowAttributes(wf, wfAttributes)
	return &defaultInspector{wf}, nil
}

// defaultInspector implements disk.Inspector using a Daisy workflow.
type defaultInspector struct {
	wf *daisy.Workflow
}

// Inspect finds partition and boot-related properties for a GCP persistent disk, and
// returns an InspectionResult. `reference` is a fully-qualified PD URI, such as
// "projects/project-name/zones/us-central1-a/disks/disk-name". `inspectOS` is a flag
// to determine whether to inspect OS on the disk.
func (inspector *defaultInspector) Inspect(reference string, inspectOS bool) (ir InspectionResult, err error) {
	inspector.wf.AddVar("pd_uri", reference)
	inspector.wf.AddVar("is_inspect_os", strconv.FormatBool(inspectOS))
	err = inspector.wf.Run(context.Background())
	if err != nil {
		return
	}

	ir.Architecture = inspector.wf.GetSerialConsoleOutputValue("architecture")
	ir.Distro = inspector.wf.GetSerialConsoleOutputValue("distro")
	ir.Major = inspector.wf.GetSerialConsoleOutputValue("major")
	ir.Minor = inspector.wf.GetSerialConsoleOutputValue("minor")

	ir.UEFIBootable, _ = strconv.ParseBool(inspector.wf.GetSerialConsoleOutputValue("uefi_bootable"))
	ir.BIOSBootable, _ = strconv.ParseBool(inspector.wf.GetSerialConsoleOutputValue("bios_bootable"))
	ir.RootFS = inspector.wf.GetSerialConsoleOutputValue("root_fs")
	return
}

func (inspector *defaultInspector) Cancel(reason string) bool {
	if inspector.wf != nil {
		inspector.wf.CancelWithReason(reason)
		return true
	}

	//indicate cancel was not performed
	return false
}

func (inspector *defaultInspector) TraceLogs() []string {
	if inspector.wf != nil && inspector.wf.Logger != nil {
		return inspector.wf.Logger.ReadSerialPortLogs()
	}
	return []string{}
}
