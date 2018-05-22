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

func TestGetChain(t *testing.T) {
	a := &Workflow{}
	b := &Workflow{parent: a}
	c := &Workflow{parent: b}
	a1 := &Step{w: a}
	a2 := &Step{w: a, IncludeWorkflow: &IncludeWorkflow{Workflow: b}}
	b1 := &Step{w: b}
	b2 := &Step{w: b, SubWorkflow: &SubWorkflow{Workflow: c}}
	c1 := &Step{w: c}
	orphan := &Step{}
	a.Steps = map[string]*Step{"a1": a1, "a2": a2}
	b.Steps = map[string]*Step{"b1": b1, "b2": b2}
	c.Steps = map[string]*Step{"c1": c1}

	tests := []struct {
		desc      string
		s         *Step
		wantChain []*Step
	}{
		{"leaf case", a1, []*Step{a1}},
		{"step from include case", b1, []*Step{a2, b1}},
		{"step from sub case", c1, []*Step{a2, b2, c1}},
		{"orphan step case", orphan, nil},
	}

	for _, tt := range tests {
		if chain := tt.s.getChain(); !reflect.DeepEqual(chain, tt.wantChain) {
			t.Errorf("%s: %v != %v", tt.desc, chain, tt.wantChain)
		}
	}
}

func TestNestedDepends(t *testing.T) {
	// root -- a1 (some step)
	//     |
	//      -- a2 (IncludeWorkflow) -- b1 (SubWorkflow) -- c1 (some step)
	//                             |
	//                              -- b2 (SubWorkflow) -- d1 (some step)
	//                             |
	//                              -- b3 (some step)
	// different root -- e1 (some step)
	a := &Workflow{}
	b := &Workflow{parent: a}
	c := &Workflow{parent: b}
	d := &Workflow{parent: b}
	e := &Workflow{}
	a1 := &Step{name: "a1", w: a}
	a2 := &Step{name: "a2", w: a, IncludeWorkflow: &IncludeWorkflow{Workflow: b}}
	b1 := &Step{name: "b1", w: b, SubWorkflow: &SubWorkflow{Workflow: c}}
	b2 := &Step{name: "b2", w: b, SubWorkflow: &SubWorkflow{Workflow: d}}
	b3 := &Step{name: "b3", w: b}
	c1 := &Step{name: "c1", w: c}
	d1 := &Step{name: "d1", w: d}
	e1 := &Step{name: "e1", w: e}
	orphan := &Step{}
	a.Steps = map[string]*Step{"a1": a1, "a2": a2}
	b.Steps = map[string]*Step{"b1": b1, "b2": b2, "b3": b3}
	c.Steps = map[string]*Step{"c1": c1}
	d.Steps = map[string]*Step{"d1": d1}
	e.Steps = map[string]*Step{"e1": e1}
	a.Dependencies = map[string][]string{"a1": {"a2"}}
	b.Dependencies = map[string][]string{"b1": {"b2"}}

	tests := []struct {
		desc   string
		s1, s2 *Step
		want   bool
	}{
		{"depends on niece/nephew case", a1, b3, true},
		{"doesn't depend on niece/nephew case", b3, c1, false},
		{"depends on great niece/nephew case", a1, c1, true},
		{"doesn't depend on son/daughter case", b1, c1, false},
		{"doesn't depend on mother/father case", c1, b1, false},
		{"depends on aunt/uncle case", c1, b2, true},
		{"depends on cousin case", c1, d1, true},
		{"doesn't depend on brother from another mother case", a1, e1, false},
		{"orphan step case", a1, orphan, false},
		{"orphan step case 2", orphan, a1, false},
		{"nil step case", a1, nil, false},
		{"nil step case 2", nil, a1, false},
	}

	for _, tt := range tests {
		got := tt.s1.nestedDepends(tt.s2)
		if got != tt.want {
			t.Errorf("%s: got %t, want %t", tt.desc, got, tt.want)
		}
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
			Step{CreateNetworks: &CreateNetworks{}},
			reflect.TypeOf(&CreateNetworks{}),
		},
		{
			Step{CopyGCSObjects: &CopyGCSObjects{}},
			reflect.TypeOf(&CopyGCSObjects{}),
		},
		{
			Step{StopInstances: &StopInstances{}},
			reflect.TypeOf(&StopInstances{}),
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
