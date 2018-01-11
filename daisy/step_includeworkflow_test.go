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

package daisy

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestIncludeWorkflowPopulate(t *testing.T) {
	// Tests:
	// - adopt parent project, zone, and gcs path
	// - included sources go into parent sources
	// - vars get passed into included workflow
	// - included workflow name is step name

	ctx := context.Background()
	w := testWorkflow()
	got := &Workflow{
		parent: w,
		Sources: map[string]string{
			"file": "path",
		},
		Steps: map[string]*Step{
			"${foo}": {
				testType: &mockStep{},
			},
		},
	}
	s := &Step{
		name: "step-name",
		w:    w,
		IncludeWorkflow: &IncludeWorkflow{
			Vars:     map[string]string{"foo": "bar"},
			Workflow: got,
		},
	}

	if err := s.IncludeWorkflow.populate(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	wantTimeout, _ := time.ParseDuration(defaultTimeout)
	want := &Workflow{
		Name:    "step-name",
		Project: w.Project,
		Zone:    w.Zone,
		GCSPath: w.GCSPath,
		id:      w.id,
		Vars: map[string]Var{
			"foo": {Value: "bar"},
		},
		autovars: map[string]string{},
		Sources: map[string]string{
			"file": "path",
		},
		Steps: map[string]*Step{
			"bar": {
				name:     "bar",
				Timeout:  defaultTimeout,
				timeout:  wantTimeout,
				testType: &mockStep{},
			},
		},
	}

	// Fixes for pretty.Compare.
	for _, wf := range []*Workflow{got, want} {
		wf.ComputeClient = nil
		wf.StorageClient = nil
		wf.Logger = nil
		wf.cleanupHooks = nil
		wf.parent = nil
		wf.gcsLogWriter = nil
		for _, s := range wf.Steps {
			s.w = nil
		}
	}

	if diffRes := diff(got, want); diffRes != "" {
		t.Errorf("populated IncludeWorkflow does not match expectation: (-got +want)\n%s", diffRes)
	}
	if diffRes := diff(w.Sources, got.Sources); diffRes != "" {
		t.Errorf("parent workflow sources don't match expectation: (-got +want)\n%s", diffRes)
	}
}

func TestIncludeWorkflowRun(t *testing.T) {}

func TestIncludeWorkflowValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	iw := w.NewIncludedWorkflow()
	dCreator, _ := w.NewStep("dCreator")
	dCreator.CreateImages = &CreateImages{}
	incStep, _ := w.NewStep("incStep")
	incStep.IncludeWorkflow = &IncludeWorkflow{Workflow: iw}
	w.AddDependency(incStep, dCreator)
	dDeleter, _ := iw.NewStep("dDeleter")
	dDeleter.DeleteResources = &DeleteResources{Disks: []string{"d"}}
	if err := disks[w].registerCreation("d", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}, dCreator, false); err != nil {
		t.Fatal(err)
	}

	if err := w.populate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := incStep.IncludeWorkflow.validate(ctx, incStep); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
