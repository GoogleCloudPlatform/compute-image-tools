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

	"fmt"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateDisksRun(t *testing.T) {
	w := testWorkflow()
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
			CreateDisks{{Disk: compute.Disk{Name: "d-foo", SourceImage: "i-dne"}}},
			nil,
			"invalid or missing reference to SourceImage \"i-dne\"",
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
		"d1": {name: "d1", real: (*cds)[0].Name, link: "link", noCleanup: false, deleted: false},
		"d2": {name: "d2", real: (*cds)[4].Name, link: "link", noCleanup: false, deleted: false},
		"d3": {name: "d3", real: (*cds)[5].Name, link: "link", noCleanup: true, deleted: false},
		"d4": {name: "d4", real: (*cds)[6].Name, link: "link", noCleanup: false, deleted: false}}

	if diff := pretty.Compare(disks[w].m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateDisksValidate(t *testing.T) {
	// Set up.
	w := testWorkflow()
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d1"}}
	validatedImages = nameSet{w: {"i1"}}

	// Good cases.
	goodTests := []struct {
		desc               string
		cd, wantCd         *CreateDisk
		wantValidatedDisks []string
	}{
		{
			"source image",
			&CreateDisk{Disk: compute.Disk{Name: "d2", SourceImage: "i1", Description: "foo"}},
			&CreateDisk{name: "d2", Disk: compute.Disk{Name: w.genName("d2"), SourceImage: "i1", Type: fmt.Sprintf("zones/%s/diskTypes/pd-standard", w.Zone), Description: "foo"}, Project: w.Project, Zone: w.Zone},
			[]string{"d1", "d2"},
		},
		{
			"blank disk",
			&CreateDisk{Disk: compute.Disk{Name: "d3", SizeGb: 50, Description: "foo"}},
			&CreateDisk{name: "d3", Disk: compute.Disk{Name: w.genName("d3"), SizeGb: 50, Type: fmt.Sprintf("zones/%s/diskTypes/pd-standard", w.Zone), Description: "foo"}, Project: w.Project, Zone: w.Zone},
			[]string{"d1", "d2", "d3"},
		},
		{
			"exact name",
			&CreateDisk{Disk: compute.Disk{Name: "d4", SourceImage: "i1", Description: "foo"}, ExactName: true},
			&CreateDisk{name: "d4", Disk: compute.Disk{Name: "d4", SourceImage: "i1", Type: fmt.Sprintf("zones/%s/diskTypes/pd-standard", w.Zone), Description: "foo"}, Project: w.Project, Zone: w.Zone, ExactName: true},
			[]string{"d1", "d2", "d3", "d4"},
		},
		{
			"non default type",
			&CreateDisk{Disk: compute.Disk{Name: "d5", SourceImage: "i1", Type: "pd-ssd", Description: "foo"}},
			&CreateDisk{name: "d5", Disk: compute.Disk{Name: w.genName("d5"), SourceImage: "i1", Type: fmt.Sprintf("zones/%s/diskTypes/pd-ssd", w.Zone), Description: "foo"}, Project: w.Project, Zone: w.Zone},
			[]string{"d1", "d2", "d3", "d4", "d5"},
		},
		{
			"non default zone",
			&CreateDisk{Disk: compute.Disk{Name: "d6", SourceImage: "i1", Description: "foo"}, Zone: "foo-zone"},
			&CreateDisk{name: "d6", Disk: compute.Disk{Name: w.genName("d6"), SourceImage: "i1", Type: "zones/foo-zone/diskTypes/pd-standard", Description: "foo"}, Project: w.Project, Zone: "foo-zone"},
			[]string{"d1", "d2", "d3", "d4", "d5", "d6"},
		},
		{
			"non default project",
			&CreateDisk{Disk: compute.Disk{Name: "d7", SourceImage: "i1", Description: "foo"}, Project: "foo-project"},
			&CreateDisk{name: "d7", Disk: compute.Disk{Name: w.genName("d7"), SourceImage: "i1", Type: fmt.Sprintf("zones/%s/diskTypes/pd-standard", w.Zone), Description: "foo"}, Project: "foo-project", Zone: w.Zone},
			[]string{"d1", "d2", "d3", "d4", "d5", "d6", "d7"},
		},
	}

	for _, tt := range goodTests {
		cds := CreateDisks{tt.cd}
		if err := cds.validate(s); err != nil {
			t.Errorf("%q: unexpected error: %v", tt.desc, err)
		}
		if diff := pretty.Compare(tt.cd, tt.wantCd); diff != "" {
			t.Errorf("%q: validated Disk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if !reflect.DeepEqual(validatedDisks[w], tt.wantValidatedDisks) {
			t.Errorf("%q: got:(%v) != want(%v)", tt.desc, validatedDisks[w], tt.wantValidatedDisks)
		}
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   *CreateDisk
		err  string
	}{
		{
			"dupe disk name",
			&CreateDisk{Disk: compute.Disk{Name: "d1", SizeGb: 50}},
			fmt.Sprintf("error adding disk: workflow %q has duplicate references for %q", w.Name, "d1"),
		},
		{
			"no size/source image",
			&CreateDisk{Disk: compute.Disk{Name: "bd1"}},
			"cannot create disk: SizeGb and SourceImage not set",
		},
		{
			"image DNE",
			&CreateDisk{Disk: compute.Disk{Name: "bd1", SourceImage: "i-dne"}},
			"cannot create disk: image not found: i-dne",
		},
	}

	for _, tt := range badTests {
		cds := CreateDisks{tt.cd}
		if err := cds.validate(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := []string{"d1", "d2", "d3", "d4", "d5", "d6", "d7"}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedDisks[w], want)
	}
}
