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

	"github.com/kylelemons/godebug/pretty"
)

func TestCreateInstancesRun(t *testing.T) {
	wf := testWorkflow()
	wf.diskRefs.m = map[string]*resource{
		"d1": {"d1", wf.genName("d1"), "link", false},
		"d2": {"d2", wf.genName("d2"), "link", false},
		"d3": {"d3", wf.genName("d3"), "link", false},
	}
	ci := &CreateInstances{
		{Name: "i1", MachineType: "foo-type", AttachedDisks: []string{"d1"}},
		{Name: "i2", MachineType: "foo-type", AttachedDisks: []string{"d2"}},
		{Name: "i3", MachineType: "foo-type", AttachedDisks: []string{"d3"}, NoCleanup: true},
		{Name: "i4", MachineType: "foo-type", AttachedDisks: []string{"d3"}, ExactName: true},
	}
	if err := ci.run(wf); err != nil {
		t.Fatalf("error running CreateInstances.run(): %v", err)
	}

	want := map[string]*resource{
		"i1": {"i1", wf.genName("i1"), "link", false},
		"i2": {"i2", wf.genName("i2"), "link", false},
		"i3": {"i3", wf.genName("i3"), "link", true},
		"i4": {"i4", "i4", "link", false},
	}

	if diff := pretty.Compare(wf.instanceRefs.m, want); diff != "" {
		t.Errorf("instanceRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateInstancesValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedDisks = nameSet{w: {"d-foo", "d-bar"}}
	validatedInstances = nameSet{w: {"i-foo"}}

	// Good case. Using multiple disks.
	ci := CreateInstances{
		CreateInstance{Name: "i-bar", AttachedDisks: []string{"d-foo", "d-bar"}},
	}
	if err := ci.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	want := []string{"i-foo", "i-bar"}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	// Good case. StartupScript.
	ci = CreateInstances{
		CreateInstance{Name: "i-bas", AttachedDisks: []string{"d-foo", "d-bar"}, StartupScript: "file"},
	}
	w.Sources = map[string]string{"file": "gs://some/file"}
	want = []string{"i-foo", "i-bar", "i-bas"}
	if err := ci.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	// Bad case. Dupe name.
	ci = CreateInstances{
		CreateInstance{Name: "i-bar", AttachedDisks: []string{"d-foo", "d-bar"}},
	}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	// Bad case. StartupScript not in Sources.
	w.Sources = nil
	ci = CreateInstances{
		CreateInstance{Name: "i-baz", AttachedDisks: []string{"d-foo", "d-bar"}, StartupScript: "file"},
	}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	// Bad case. No disks.
	ci = CreateInstances{CreateInstance{Name: "i-baz"}}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	// Bad case. Disk DNE.
	ci = CreateInstances{
		CreateInstance{Name: "i-baz", AttachedDisks: []string{"d-foo", "d-bar", "d-dne"}},
	}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedInstances[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedInstances[w], want)
	}
}
