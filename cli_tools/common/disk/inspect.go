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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/daisycommon"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

const (
	workflowFile = "image_import/inspection/inspect-disk.wf.json"
)

// Inspector finds partition and boot-related properties for a disk.
type Inspector interface {
	// Inspect finds partition and boot-related properties for a disk and
	// returns an InspectionResult. The reference is implementation specific.
	Inspect(reference string) (InspectionResult, error)
}

// InspectionResult contains the partition and boot-related properties of a disk.
type InspectionResult struct {
}

// NewInspector creates an Inspector that can inspect GCP disks.
func NewInspector(wfAttributes daisycommon.WorkflowAttributes) (Inspector, error) {
	wf, err := daisy.NewFromFile(path.Join(wfAttributes.WorkflowDirectory, workflowFile))
	if err != nil {
		return nil, err
	}
	daisycommon.SetWorkflowAttributes(wf, wfAttributes)
	return defaultInspector{wf}, nil
}

// defaultInspector implements disk.Inspector using a Daisy workflow.
type defaultInspector struct {
	wf *daisy.Workflow
}

// Inspect finds partition and boot-related properties for a GCP persistent disk, and
// returns an InspectionResult. `reference` is a fully-qualified PD URI, such as
// "projects/project-name/zones/us-central1-a/disks/disk-name".
func (inspector defaultInspector) Inspect(reference string) (InspectionResult, error) {
	inspector.wf.AddVar("pd_uri", reference)
	err := inspector.wf.Run(context.Background())
	return InspectionResult{}, err
}
