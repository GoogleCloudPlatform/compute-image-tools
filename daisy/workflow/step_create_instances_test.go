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

func TestCreateInstancesRun(t *testing.T) {
	wf := testWorkflow()
	wf.diskRefs = map[string]string{
		namer("disk1", testWf, testSuffix): "link",
		namer("disk2", testWf, testSuffix): "link",
		namer("disk3", testWf, testSuffix): "link"}
	ci := &CreateInstances{
		{Name: "instance1", MachineType: "foo-type", AttachedDisks: []string{"disk1"}},
		{Name: "instance2", MachineType: "foo-type", AttachedDisks: []string{"disk2"}},
		{Name: "instance3", MachineType: "foo-type", AttachedDisks: []string{"disk3"}}}
	if err := ci.run(wf); err != nil {
		t.Fatalf("error running CreateInstances.run(): %v", err)
	}

	want := []string{
		namer("instance1", testWf, testSuffix),
		namer("instance2", testWf, testSuffix),
		namer("instance3", testWf, testSuffix)}

	for _, name := range wf.instanceRefs {
		if !containsString(name, want) {
			t.Errorf("Workflow.createdInstances does not contain expected instance %s", name)
		}
	}

	for _, name := range want {
		if !containsString(name, wf.instanceRefs) {
			t.Errorf("Workflow.createdInstances does not contain expected instance %s", name)
		}
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
		t.Error("validation should not have failed")
	}
	if !reflect.DeepEqual(validatedInstances, nameSet{w: {"i-foo", "i-bar"}}) {
		t.Errorf("%v != %v", validatedInstances, nameSet{w: {"i-foo", "i-bar"}})
	}

	// Bad case. Dupe name.
	ci = CreateInstances{
		CreateInstance{Name: "i-bar", AttachedDisks: []string{"d-foo", "d-bar"}},
	}
	if !reflect.DeepEqual(validatedInstances, nameSet{w: {"i-foo", "i-bar"}}) {
		t.Errorf("%v != %v", validatedInstances, nameSet{w: {"i-foo", "i-bar"}})
	}

	// Bad case. No disks.
	ci = CreateInstances{CreateInstance{Name: "i-baz"}}
	if err := ci.validate(w); err == nil {
		t.Error("validation should have failed")
	}
	if !reflect.DeepEqual(validatedInstances, nameSet{w: {"i-foo", "i-bar"}}) {
		t.Errorf("%v != %v", validatedInstances, nameSet{w: {"i-foo", "i-bar"}})
	}

	// Bad case. Disk DNE.
	ci = CreateInstances{
		CreateInstance{Name: "i-baz", AttachedDisks: []string{"d-foo", "d-bar", "d-dne"}},
	}
	if err := ci.validate(w); err == nil {
		t.Error("validation should have failed")
	}
	if !reflect.DeepEqual(validatedInstances, nameSet{w: {"i-foo", "i-bar"}}) {
		t.Errorf("%v != %v", validatedInstances, nameSet{w: {"i-foo", "i-bar"}})
	}
}
