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
	"github.com/kylelemons/godebug/pretty"
	"testing"
	"time"
	"context"
)

func TestIncludeWorkflowPopulate(t *testing.T) {
	// Tests:
	// - adopt parent project, zone, and gcs path
	// - included sources go into parent sources
	// - vars get passed into included workflow
	// - included workflow name is step name

	ctx:=context.Background()
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
			Vars: map[string]string{"foo": "bar"},
			w:    got,
		},
	}

	if err := s.IncludeWorkflow.populate(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	wantTimeout, _ := time.ParseDuration(defaultTimeout)
	want := &Workflow{
		Name:    s.name,
		Project: w.Project,
		Zone:    w.Zone,
		GCSPath: w.GCSPath,
		Vars: map[string]vars{
			"foo": {Value: "bar"},
		},
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
		wf.logger = nil
		wf.cleanupHooks = nil
		wf.parent = nil
		for _, s := range wf.Steps {
			s.w = nil
		}
	}

	if diff := pretty.Compare(got, want); diff != "" {
		t.Errorf("populated IncludeWorkflow does not match expectation: (-got +want)\n%s", diff)
	}
	if diff := pretty.Compare(w.Sources, got.Sources); diff != "" {
		t.Errorf("parent workflow sources don't match expectation: (-got +want)\n%s", diff)
	}
}

func TestIncludeWorkflowRun(t *testing.T) {}

func TestIncludeWorkflowValidate(t *testing.T) {
	ctx:=context.Background()
	w := testWorkflow()
	disks[w].add("foo", &resource{})
	iw := w.NewIncludedWorkflow()
	w.Steps = map[string]*Step{
		"included": {
			IncludeWorkflow: &IncludeWorkflow{
				w: iw,
			},
		},
	}
	iw.Steps = map[string]*Step{
		"del": {
			DeleteResources: &DeleteResources{
				Disks: []string{"foo"},
			},
		},
	}

	w.populate(ctx)
	s := w.Steps["included"]
	if err := s.IncludeWorkflow.populate(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
