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
	"errors"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateDisksRun(t *testing.T) {
	w := testWorkflow()
	var fakeClientErr error
	w.ComputeClient.CreateDiskFake = func(project, zone string, d *compute.Disk) error {
		if fakeClientErr != nil {
			return fakeClientErr
		}
		d.SelfLink = "link"
		return nil
	}
	s := &Step{w: w}
	images[w].m = map[string]*resource{"i1": {"i1", w.genName("i1"), "link", false, false}}

	cds := &CreateDisks{
		{name: "d1", Disk: compute.Disk{Name: w.genName("d1"), SourceImage: "i1", SizeGb: 100, Type: ""}},
		{name: "d2", Disk: compute.Disk{Name: w.genName("d2"), SourceImage: "projects/project/global/images/family/my-family", SizeGb: 100, Type: ""}},
		{name: "d2", Disk: compute.Disk{Name: w.genName("d2"), SourceImage: "projects/project/global/images/i2", SizeGb: 100, Type: ""}},
		{name: "d2", Disk: compute.Disk{Name: w.genName("d2"), SourceImage: "global/images/family/my-family", SizeGb: 100, Type: ""}},
		{name: "d2", Disk: compute.Disk{Name: w.genName("d2"), SourceImage: "global/images/i2", SizeGb: 100, Type: ""}, Zone: "zone", Project: "project"},
		{name: "d3", Disk: compute.Disk{Name: w.genName("d3"), SourceImage: "i1", SizeGb: 100, Type: ""}, NoCleanup: true},
		{name: "d4", Disk: compute.Disk{Name: "d4", SourceImage: "i1", SizeGb: 100, Type: ""}}}
	if err := cds.run(s); err != nil {
		t.Errorf("error running CreateDisks.run(): %v", err)
	}

	// Bad cases.
	badTests := []struct {
		name          string
		cd            CreateDisks
		fakeClientErr error
		err           string
	}{
		{
			"image DNE",
			CreateDisks{CreateDisk{Disk: compute.Disk{Name: "d-foo", SourceImage: "i-dne"}}},
			nil,
			"invalid or missing reference to SourceImage \"i-dne\"",
		},
		{
			"client failure",
			CreateDisks{CreateDisk{}},
			errors.New("client err"),
			"client err",
		},
	}

	for _, tt := range badTests {
		fakeClientErr = tt.fakeClientErr
		if err := tt.cd.run(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := map[string]*resource{
		"d1": {name: "d1", real: (*cds)[0].Disk.Name, link: "link", noCleanup: false, deleted: false},
		"d2": {name: "d2", real: (*cds)[4].Disk.Name, link: "link", noCleanup: false, deleted: false},
		"d3": {name: "d3", real: (*cds)[5].Disk.Name, link: "link", noCleanup: true, deleted: false},
		"d4": {name: "d4", real: (*cds)[6].Disk.Name, link: "link", noCleanup: false, deleted: false}}

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
			CreateDisks{{Disk: compute.Disk{Name: "d-bar", SourceImage: "i-foo", SizeGb: 50}}},
			[]string{"d-foo", "d-bar"},
		},
		{
			"blank disk",
			CreateDisks{{Disk: compute.Disk{Name: "d-baz", SizeGb: 50}, Zone: "foo"}},
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
			CreateDisks{{Disk: compute.Disk{Name: "d-foo", SizeGb: 50}}},
			"error adding disk: workflow \"\" has duplicate references for \"d-foo\"",
		},
		{
			"no Size.",
			CreateDisks{{Disk: compute.Disk{Name: "d-foo"}}},
			"cannot create disk: SizeGb and SourceImage not set",
		},
		{
			"image DNE",
			CreateDisks{{Disk: compute.Disk{Name: "d-foo", SourceImage: "i-dne"}}},
			"cannot create disk: image not found: i-dne",
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
