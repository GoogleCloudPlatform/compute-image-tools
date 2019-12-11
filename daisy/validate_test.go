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
	"reflect"
	"sync"
	"testing"
)

func TestCheckName(t *testing.T) {
	// Test a good name passes.
	goodName := "this-is-a-g00d-name"
	if !checkName(goodName) {
		t.Errorf("%s incorrectly labeled as a bad name", goodName)
	}

	// Test bad names fail.
	badNames := []string{
		"This-is-a-bad-name",
		"this-IS-a-bad-name",
		"this_is_a_bad_name",
		"-this-is-a-bad-name",
		"2this-is-a-bad-name",
		"this-is-@-bad-name",
	}
	for _, badName := range badNames {
		if checkName(badName) {
			t.Errorf("%s incorrectly labeled as a good name", badName)
		}
	}
}

func TestValidateVarsSubbed(t *testing.T) {
	w := testWorkflow()

	if err := w.validateVarsSubbed(); err != nil {
		t.Errorf("unexpected error on good workflow: %s", err)
	}

	w.Name = "workflow-${unsubbed}"
	want := `Unresolved var "${unsubbed}" found in "workflow-${unsubbed}"`
	if err := w.validateVarsSubbed(); err.Error() != want {
		t.Errorf("workflow with unsubbed var bad error, want: %q got: %q", want, err.Error())
	}

	//Workflow.RequiredVars = []string{"unsubbed"}
	//want = `Unresolved required var "${unsubbed}" found in "workflow-${unsubbed}"`
	//if err := Workflow.validateVarsSubbed(); err.Error() != want {
	//	t.Errorf("workflow with unsubbed required var bad error, want: %q got: %q", want, err.Error())
	//}
}

func TestValidateWorkflow(t *testing.T) {
	ctx := context.Background()
	// Normal, good validation.
	w := testWorkflow()
	s := &Step{Timeout: "10s", testType: &mockStep{}, w: w}
	w.Steps = map[string]*Step{"s0": s}
	if err := w.populate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.validate(ctx); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	logger := &MockLogger{}
	// Bad test cases.
	tests := []struct {
		desc string
		wf   *Workflow
	}{
		{"no name", &Workflow{Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Logger: logger}},
		{"no project", &Workflow{Name: "n", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Logger: logger}},
		{"no zone", &Workflow{Name: "n", Project: "p", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": s}, Logger: logger}},
		{"no steps", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Logger: logger}},
		{"no step name", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"": s}, Logger: logger}},
		{"no step type", &Workflow{Name: "n", Project: "p", Zone: "z", GCSPath: "b", OAuthPath: "o", Steps: map[string]*Step{"s": {Timeout: defaultTimeout, w: w}}, Logger: logger}},
	}

	for _, tt := range tests {
		tt.wf.Cancel = make(chan struct{})
		if err := tt.wf.Validate(ctx); err == nil {
			t.Errorf("validation should have failed on %v because of %q", tt.wf, tt.desc)
		}
	}
}

func TestValidateDAG(t *testing.T) {
	ctx := context.Background()
	calls := make([]int, 5)
	errs := make([]DError, 5)
	var rw sync.Mutex
	mockValidate := func(i int) func(ctx context.Context, s *Step) DError {
		return func(ctx context.Context, s *Step) DError {
			rw.Lock()
			defer rw.Unlock()
			calls[i] = calls[i] + 1
			return errs[i]
		}
	}
	reset := func() {
		rw.Lock()
		defer rw.Unlock()
		calls = make([]int, 5)
		errs = make([]DError, 5)
	}

	// s0---->s1---->s3
	//   \         /
	//    --->s2---
	// s4
	w := testWorkflow()
	w.Steps = map[string]*Step{
		"s0": {testType: &mockStep{validateImpl: mockValidate(0)}, w: w},
		"s1": {testType: &mockStep{validateImpl: mockValidate(1)}, w: w},
		"s2": {testType: &mockStep{validateImpl: mockValidate(2)}, w: w},
		"s3": {testType: &mockStep{validateImpl: mockValidate(3)}, w: w},
		"s4": {testType: &mockStep{validateImpl: mockValidate(4)}, w: w},
	}
	w.Dependencies = map[string][]string{
		"s1": {"s0"},
		"s2": {"s0", "s0"}, // Check that dupes are removed.
		"s3": {"s1", "s2"},
	}

	// Normal case -- no issues.
	if err := w.populate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.validateDAG(ctx); err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	for i, callCount := range calls {
		if callCount != 1 {
			t.Errorf("step %d did not get validated", i)
		}
	}
	if !reflect.DeepEqual(w.Dependencies["s2"], []string{"s0"}) {
		t.Error("duplicate dependency not removed")
	}

	// Reset.
	reset()

	// Failed step 2.
	errs[2] = Errf("fail")
	if err := w.validateDAG(ctx); err == nil {
		t.Error("step 2 should have failed validation")
	}

	// Reset.
	reset()

	// Fail, missing dep.
	w.Dependencies["s0"] = []string{"dne"}
	if err := w.validateDAG(ctx); err == nil {
		t.Error("validation should have failed due to missing dependency")
	}

	// Fail, missing step.
	w.Dependencies["dne"] = []string{"s0"}
	if err := w.validateDAG(ctx); err == nil {
		t.Error("validation should have failed due to missing dependency")
	}

	// Fail, cyclical deps.
	w.Dependencies["s0"] = []string{"s3"}
	if err := w.validateDAG(ctx); err == nil {
		t.Error("validation should have failed due to dependency cycle")
	}
}
