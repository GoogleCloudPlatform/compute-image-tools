//  Copyright 2017 Google Inc. All Rights Reserved.
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

package workflow

import (
	"reflect"
	"testing"
)

func testSubWorkflowSetup() (*Workflow, *Step, error) {
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"disks": {
			CreateDisks: &CreateDisks{
				{
					Name:        "disk4",
					SourceImage: "",
					SizeGB:      "50",
				},
			},
		},
		"sub workflow": {
			SubWorkflow: &SubWorkflow{
				Path: "cloud/cluster/guest/daisy/workflow/test_sub.workflow",
				Workflow: &Workflow{
					Steps: map[string]*Step{
						"disks": {
							CreateDisks: &CreateDisks{
								{
									Name:        "disk1",
									SourceImage: "",
									SizeGB:      "50",
								},
								{
									Name:        "disk2",
									SourceImage: "",
									SizeGB:      "50",
								},
								{
									Name:        "disk3",
									SourceImage: "",
									SizeGB:      "50",
								},
							},
						},
						"instances": {
							CreateInstances: &CreateInstances{
								{
									Name:          "instance1",
									AttachedDisks: []string{"disk1"},
									MachineType:   "foo-type",
								},
								{
									Name:          "instance2",
									AttachedDisks: []string{"disk2"},
									MachineType:   "foo-type",
								},
								{
									Name:          "instance3",
									AttachedDisks: []string{"disk3"},
									MachineType:   "foo-type",
								},
							},
						},
						"images": {
							CreateImages: &CreateImages{
								{Name: "image1", SourceDisk: "disk1"},
								{Name: "image2", SourceDisk: "disk2"},
								{Name: "image3", SourceDisk: "disk3"},
								{Name: "image4", SourceDisk: "disk4"},
							},
						},
						"delete": {
							DeleteResources: &DeleteResources{
								Instances: []string{"instance2"},
								Images:    []string{"image3"},
								Disks:     []string{"disk1"},
							},
						},
					},
					Dependencies: map[string][]string{
						"disks":     {},
						"instances": {"disks"},
						"images":    {"instances"},
						"delete":    {"images"},
					},
				},
			},
		},
	}

	w.Dependencies = map[string][]string{
		"disks":        {},
		"sub workflow": {"disks"},
	}

	for _, step := range w.Steps {
		if err := w.populateStep(step); err != nil {
			return nil, nil, err
		}
	}
	w.createdDisks = map[string]string{namer("disk4", testWf, testSuffix): "link"}

	return w, w.Steps["sub workflow"], nil
}

func TestSubWorkflowRun(t *testing.T) {
	w, step, err := testSubWorkflowSetup()
	if err != nil {
		t.Fatalf("error running testSubWorkflowSetup(): %v", err)
	}

	if err := step.run(w); err != nil {
		t.Fatalf("error running SubWorkflow.run(): %v", err)
	}

	wDisks := map[string]string{
		namer("disk2", testWf, testSuffix): "link",
		namer("disk3", testWf, testSuffix): "link",
		namer("disk4", testWf, testSuffix): "link",
	}
	if !reflect.DeepEqual(w.createdDisks, wDisks) {
		t.Errorf("Workflow.createdDisks does not match expectations, got: %q, want: %q", w.createdDisks, wDisks)
	}

	wInstances := []string{
		namer("instance1", testWf, testSuffix),
		namer("instance3", testWf, testSuffix),
	}
	for _, name := range w.createdInstances {
		if !containsString(name, wInstances) {
			t.Errorf("Workflow.createdInstances does not contain expected instance %s", name)
		}
	}

	wImages := map[string]string{
		"image1": "link",
		"image2": "link",
		"image4": "link",
	}
	if !reflect.DeepEqual(w.createdImages, wImages) {
		t.Errorf("Workflow.createdImages does not match expectations, got: %q, want: %q", w.createdImages, wImages)
	}
}

func TestSubWorkflowValidate(t *testing.T) {
	w, _, err := testSubWorkflowSetup()
	if err != nil {
		t.Fatalf("error running testSubWorkflowSetup(): %v", err)
	}

	// We don't call step.validate() directly as it can't validate properly
	// without the surrounding workflow.
	if err := w.validate(); err != nil {
		t.Errorf("SubWorkflow validation failed: %v", err)
	}
}
