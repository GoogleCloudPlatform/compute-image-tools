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

func TestCreateDisksRun(t *testing.T) {
	wf := testWorkflow()
	wf.imageRefs.m = map[string]*resource{"i1": {"i1", wf.genName("i1"), "link", false}}
	cd := &CreateDisks{
		{Name: "d1", SourceImage: "i1", SizeGB: "100", Type: ""},
		{Name: "d2", SourceImage: "projects/global/images/i2", SizeGB: "100", Type: ""},
		{Name: "d3", SourceImage: "i1", SizeGB: "100", Type: "", NoCleanup: true},
		{Name: "d4", SourceImage: "i1", SizeGB: "100", Type: "", ExactName: true}}
	if err := cd.run(wf); err != nil {
		t.Fatalf("error running CreateDisks.run(): %v", err)
	}

	want := map[string]*resource{
		"d1": {"d1", wf.genName("d1"), "link", false},
		"d2": {"d2", wf.genName("d2"), "link", false},
		"d3": {"d3", wf.genName("d3"), "link", true},
		"d4": {"d4", "d4", "link", false}}

	if diff := pretty.Compare(wf.diskRefs.m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
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
		t.Errorf("validation should not have failed: %v", err)
	}
	want := []string{"d-foo", "d-bar"}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedDisks[w], want)
	}

	// Good case. No source image.
	cd = CreateDisks{CreateDisk{Name: "d-baz", SizeGB: "50"}}
	if err := cd.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	want = []string{"d-foo", "d-bar", "d-baz"}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedDisks[w], want)
	}

	// Bad case. Dupe disk name.
	cd = CreateDisks{CreateDisk{Name: "d-foo", SizeGB: "50"}}
	if err := cd.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedDisks[w], want)
	}

	// Bad case. No Size.
	cd = CreateDisks{CreateDisk{Name: "d-new"}}
	if err := cd.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedDisks[w], want)
	}

	// Bad case. Image DNE.
	cd = CreateDisks{CreateDisk{Name: "d-gaz", SourceImage: "i-dne"}}
	if err := cd.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedDisks[w], want)
	}
}
