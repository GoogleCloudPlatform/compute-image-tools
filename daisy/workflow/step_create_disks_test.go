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
	w := testWorkflow()
	s := &Step{w: w}
	images[w].m = map[string]*resource{"i1": {"i1", w.genName("i1"), "link", false, false}}
	cd := &CreateDisks{
		{Name: "d1", SourceImage: "i1", SizeGB: "100", Type: ""},
		{Name: "d2", SourceImage: "projects/project/global/images/family/my-family", SizeGB: "100", Type: ""},
		{Name: "d2", SourceImage: "projects/project/global/images/i2", SizeGB: "100", Type: ""},
		{Name: "d2", SourceImage: "global/images/family/my-family", SizeGB: "100", Type: ""},
		{Name: "d2", SourceImage: "global/images/i2", SizeGB: "100", Type: "", Zone: "zone", Project: "project"},
		{Name: "d3", SourceImage: "i1", SizeGB: "100", Type: "", NoCleanup: true},
		{Name: "d4", SourceImage: "i1", SizeGB: "100", Type: "", ExactName: true}}
	if err := cd.run(s); err != nil {
		t.Errorf("error running CreateDisks.run(): %v", err)
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   CreateDisks
		err  string
	}{
		{
			"image DNE",
			CreateDisks{CreateDisk{Name: "d-foo", SourceImage: "i-dne"}},
			"invalid or missing reference to SourceImage \"i-dne\"",
		},
		{
			"bad size",
			CreateDisks{CreateDisk{Name: "d-foo", SizeGB: "50s"}},
			"strconv.ParseInt: parsing \"50s\": invalid syntax",
		},
	}

	for _, tt := range badTests {
		if err := tt.cd.run(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := map[string]*resource{
		"d1": {"d1", w.genName("d1"), "link", false, false},
		"d2": {"d2", w.genName("d2"), "link", false, false},
		"d3": {"d3", w.genName("d3"), "link", true, false},
		"d4": {"d4", "d4", "link", false, false}}

	if diff := pretty.Compare(disks[w].m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateDisksValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d-foo"}}
	validatedImages = nameSet{w: {"i-foo"}}

	// Good cases.
	goodTests := []struct {
		name string
		cd   CreateDisks
		want []string
	}{
		{
			"source image",
			CreateDisks{{Name: "d-bar", SourceImage: "i-foo", SizeGB: "50"}},
			[]string{"d-foo", "d-bar"},
		},
		{
			"blank disk",
			CreateDisks{{Name: "d-baz", SizeGB: "50", Zone: "foo"}},
			[]string{"d-foo", "d-bar", "d-baz"},
		},
	}

	for _, tt := range goodTests {
		if err := tt.cd.validate(s); err != nil {
			t.Errorf("%q: unexpected error: %v", tt.name, err)
		}
		if !reflect.DeepEqual(validatedDisks[w], tt.want) {
			t.Errorf("%q: got:(%v) != want(%v)", tt.name, validatedDisks[w], tt.want)
		}
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   CreateDisks
		err  string
	}{
		{
			"dupe disk name",
			CreateDisks{{Name: "d-foo", SizeGB: "50"}},
			"error adding disk: workflow \"\" has duplicate references for \"d-foo\"",
		},
		{
			"no Size.",
			CreateDisks{{Name: "d-foo"}},
			"cannot create disk: SizeGB and SourceImage not set: ",
		},
		{
			"image DNE",
			CreateDisks{{Name: "d-foo", SourceImage: "i-dne"}},
			"cannot create disk: image not found: i-dne",
		},
		{
			"bad size",
			CreateDisks{{Name: "d-foo", SizeGB: "50s"}},
			"cannot parse SizeGB: 50s, err: strconv.ParseInt: parsing \"50s\": invalid syntax",
		},
	}

	for _, tt := range badTests {
		if err := tt.cd.validate(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := []string{"d-foo", "d-bar", "d-baz"}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedDisks[w], want)
	}
}
