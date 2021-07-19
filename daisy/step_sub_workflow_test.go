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
	"reflect"
	"testing"
)

func TestSubWorkflowPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.populate(ctx)
	sw := w.NewSubWorkflow()
	sw.Vars = map[string]Var{"foo": {Value: "bar1"}, "baz": {Value: "gaz"}}
	s := &Step{
		name: "sw-step",
		w:    w,
		SubWorkflow: &SubWorkflow{
			Vars:     map[string]string{"foo": "bar2"},
			Workflow: sw,
		},
	}
	if err := w.populateStep(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if sw.Name != s.name {
		t.Errorf("unexpected subworkflow Name: %q != %q", sw.Name, s.name)
	}
	if sw.Project != w.Project {
		t.Errorf("unexpected subworkflow Project: %q != %q", sw.Project, w.Project)
	}
	if sw.Zone != w.Zone {
		t.Errorf("unexpected subworkflow Zone: %q != %q", sw.Zone, w.Zone)
	}
	wantGCSPath := fmt.Sprintf("gs://%s/%s", w.bucket, w.scratchPath)
	if sw.GCSPath != wantGCSPath {
		t.Errorf("unexpected subworkflow GCSPath: %q != %q", sw.GCSPath, wantGCSPath)
	}
	wantVars := map[string]Var{"foo": {Value: "bar2"}, "baz": {Value: "gaz"}}
	if !reflect.DeepEqual(sw.Vars, wantVars) {
		t.Errorf("unexpected subworkflow Vars: %v != %v", sw.Vars, wantVars)
	}
}

func TestSubWorkflowPopulate_SkipsReadingPathWhenWorkflowNil(t *testing.T) {
	child := testWorkflow()
	parent := testWorkflow()
	parent.Steps = map[string]*Step{
		"child": {
			SubWorkflow: &SubWorkflow{
				Path:     "test-will-fail-if-this-is-read",
				Workflow: child,
			},
		},
	}
	if err := parent.populate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSubWorkflowRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.populate(ctx)
	sw := w.NewSubWorkflow()
	s := &Step{
		name: "sw-step",
		w:    w,
		SubWorkflow: &SubWorkflow{
			Workflow: sw,
		},
	}
	if err := w.populateStep(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if err := s.SubWorkflow.run(ctx, s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSubWorkflowValidate(t *testing.T) {}
