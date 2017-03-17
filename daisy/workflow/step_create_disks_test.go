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

func TestCreateDisksRun(t *testing.T) {
	wf := testWorkflow()
	cd := &CreateDisks{
		{Name: "disk1", SourceImage: "", SizeGB: "100", SSD: false},
		{Name: "disk2", SourceImage: "", SizeGB: "100", SSD: false},
		{Name: "disk3", SourceImage: "", SizeGB: "100", SSD: false}}
	if err := cd.run(wf); err != nil {
		t.Fatalf("error running CreateDisks.run(): %v", err)
	}

	want := map[string]string{
		namer("disk1", testWf, testSuffix): "link",
		namer("disk2", testWf, testSuffix): "link",
		namer("disk3", testWf, testSuffix): "link"}
	if !reflect.DeepEqual(wf.createdDisks, want) {
		t.Errorf("Workflow.createdDisks does not match expectations, got: %q, want: %q", wf.createdDisks, want)
	}
}

func TestCreateDisksValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedDisks = nameSet{w: {"d-foo"}}
	validatedImages = nameSet{w: {"i-foo"}}

	// Good case.
	cd := CreateDisks{CreateDisk{Name: "d-bar", SourceImage: "i-foo", SizeGB: "50"}}
	if err := cd.validate(w); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(validatedDisks, nameSet{w: {"d-foo", "d-bar"}}) {
		t.Errorf("%v != %v", validatedDisks, nameSet{w: {"d-foo", "d-bar"}})
	}

	// Good case. No source image.
	cd = CreateDisks{CreateDisk{Name: "d-baz", SizeGB: "50"}}
	if err := cd.validate(w); err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}}) {
		t.Errorf("%v != %v", validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}})
	}

	// Bad case. Dupe disk name.
	cd = CreateDisks{CreateDisk{Name: "d-foo", SizeGB: "50"}}
	if err := cd.validate(w); err == nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}}) {
		t.Errorf("%v != %v", validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}})
	}

	// Bad case. No Size.
	cd = CreateDisks{CreateDisk{Name: "d-new"}}
	if err := cd.validate(w); err == nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}}) {
		t.Errorf("%v != %v", validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}})
	}

	// Bad case. Image DNE.
	cd = CreateDisks{CreateDisk{Name: "d-gaz", SourceImage: "i-dne"}}
	if err := cd.validate(w); err == nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}}) {
		t.Errorf("%v != %v", validatedDisks, nameSet{w: {"d-foo", "d-bar", "d-baz"}})
	}
}
