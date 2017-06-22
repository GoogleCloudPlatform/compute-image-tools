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

func TestDepends(t *testing.T) {
	w := &Workflow{Dependencies: map[string][]string{}}
	s1 := &Step{name: "s1", w: w}
	s2 := &Step{name: "s2", w: w}
	s3 := &Step{name: "s3", w: w}
	s4 := &Step{name: "s4", w: w}
	s5 := &Step{name: "s5", w: w}
	w.Steps = map[string]*Step{"s1": s1, "s2": s2, "s3": s3, "s4": s4, "s5": s5}

	// Check proper false.
	if s1.depends(s2) {
		t.Error("s1 shouldn't depend on s2")
	}

	// Check proper true.
	w.Dependencies["s1"] = []string{"s2"}
	if !s1.depends(s2) {
		t.Error("s1 should depend on s2")
	}

	// Check transitive dependency returns true.
	w.Dependencies["s2"] = []string{"s3"}
	if !s1.depends(s3) {
		t.Error("s1 should transitively depend on s3")
	}

	// Check cyclical graph terminates.
	w.Dependencies["s2"] = append(w.Dependencies["s2"], "s4")
	w.Dependencies["s4"] = []string{"s2"}
	// s1 doesn't have any relation to s5, but we need to check
	// if this can terminate on graphs with cycles.
	if s1.depends(s5) {
		t.Error("s1 shouldn't depend on s5")
	}

	// Check self depends on self -- false case.
	if s1.depends(s1) {
		t.Error("s1 shouldn't depend on s1")
	}

	// Check self depends on self true.
	w.Dependencies["s5"] = []string{"s5"}
	if !s5.depends(s5) {
		t.Error("s5 should depend on s5")
	}
}

func TestStepImpl(t *testing.T) {
	// Good. Try normal, working case.
	tests := []struct {
		step     Step
		stepType reflect.Type
	}{
		{
			Step{CreateDisks: &CreateDisks{}},
			reflect.TypeOf(&CreateDisks{}),
		},
		{
			Step{CreateImages: &CreateImages{}},
			reflect.TypeOf(&CreateImages{}),
		},
		{
			Step{CreateInstances: &CreateInstances{}},
			reflect.TypeOf(&CreateInstances{}),
		},
		{
			Step{CopyGCSObjects: &CopyGCSObjects{}},
			reflect.TypeOf(&CopyGCSObjects{}),
		},
		{
			Step{DeleteResources: &DeleteResources{}},
			reflect.TypeOf(&DeleteResources{}),
		},
		{
			Step{IncludeWorkflow: &IncludeWorkflow{}},
			reflect.TypeOf(&IncludeWorkflow{}),
		},
		{
			Step{SubWorkflow: &SubWorkflow{}},
			reflect.TypeOf(&SubWorkflow{}),
		},
		{
			Step{WaitForInstancesSignal: &WaitForInstancesSignal{}},
			reflect.TypeOf(&WaitForInstancesSignal{}),
		},
	}

	for _, tt := range tests {
		st, err := tt.step.stepImpl()
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		got := reflect.TypeOf(st)
		if got != tt.stepType {
			t.Errorf("unexpected step type, want: %s, got: %s", tt.stepType, got)
		}
	}

	// Bad. Try empty step.
	s := Step{}
	if _, err := s.stepImpl(); err == nil {
		t.Fatal("empty step should have thrown an error")
	}
	// Bad. Try step with multiple real steps.
	s = Step{
		CreateDisks:  &CreateDisks{},
		CreateImages: &CreateImages{},
	}
	if _, err := s.stepImpl(); err == nil {
		t.Fatal("malformed step should have thrown an error")
	}
}
