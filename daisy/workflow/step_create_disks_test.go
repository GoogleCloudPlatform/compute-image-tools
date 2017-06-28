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
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateDisksPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.ComputeClient = nil
	w.StorageClient = nil
	s := &Step{w: w}

	genFoo := w.genName("foo")
	defType := fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", w.Project, w.Zone)
	tests := []struct {
		desc        string
		input, want *CreateDisk
		wantErr     bool
	}{
		{
			"defaults case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"nondefaults case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: "pd-ssd"}, SizeGb: "10", Project: "pfoo", Zone: "zfoo"},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: "projects/pfoo/zones/zfoo/diskTypes/pd-ssd", SizeGb: 10}, daisyName: "foo", SizeGb: "10", Project: "pfoo", Zone: "zfoo"},
			false,
		},
		{
			"ExactName case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}, ExactName: true},
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone, ExactName: true},
			false,
		},
		{
			"extend Type URL case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", Type: "zones/zfoo/diskTypes/pd-ssd"}, Project: "pfoo"},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: "projects/pfoo/zones/zfoo/diskTypes/pd-ssd"}, daisyName: "foo", Project: "pfoo", Zone: w.Zone},
			false,
		},
		{
			"extend SourceImage URL case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"SourceImage daisy name case",
			&CreateDisk{Disk: compute.Disk{Name: "foo", SourceImage: "ifoo"}},
			&CreateDisk{Disk: compute.Disk{Name: genFoo, SourceImage: "ifoo", Type: defType}, daisyName: "foo", Project: w.Project, Zone: w.Zone},
			false,
		},
		{
			"bad SizeGb case",
			&CreateDisk{Disk: compute.Disk{Name: "foo"}, SizeGb: "ten"},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		cds := &CreateDisks{tt.input}
		err := cds.populate(ctx, s)
		// Short circuit the description field -- difficult to test, and unimportant.
		if tt.want != nil {
			tt.want.Description = tt.input.Description
		}
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: should have returned an error but didn't", tt.desc)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if diff := pretty.Compare(tt.input, tt.want); diff != "" {
			t.Errorf("%s: populated CreateDisk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
	}
}

func TestCreateDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	images[w].m = map[string]*resource{"i1": {real: "i1", link: "link"}}

	cds := &CreateDisks{
		{daisyName: "d1", Disk: compute.Disk{Name: "d1", SourceImage: "i1", SizeGb: 100, Type: ""}},
		{daisyName: "d2", Disk: compute.Disk{Name: "d2", SourceImage: "projects/project/global/images/family/my-family", SizeGb: 100, Type: ""}},
		{daisyName: "d2", Disk: compute.Disk{Name: "d2", SourceImage: "projects/project/global/images/i2", SizeGb: 100, Type: ""}},
		{daisyName: "d2", Disk: compute.Disk{Name: "d2", SourceImage: "global/images/family/my-family", SizeGb: 100, Type: ""}},
		{daisyName: "d2", Disk: compute.Disk{Name: "d2", SourceImage: "global/images/i2", SizeGb: 100, Type: ""}, Zone: "zone", Project: "project"},
		{daisyName: "d3", Disk: compute.Disk{Name: "d3", SourceImage: "i1", SizeGb: 100, Type: ""}, NoCleanup: true},
		{daisyName: "d4", Disk: compute.Disk{Name: "d4", SourceImage: "i1", SizeGb: 100, Type: ""}}}
	if err := cds.run(ctx, s); err != nil {
		t.Errorf("error running CreateDisks.run(): %v", err)
	}

	want := map[string]*resource{
		"d1": {real: (*cds)[0].Name, link: "link", noCleanup: false},
		"d2": {real: (*cds)[4].Name, link: "link", noCleanup: false},
		"d3": {real: (*cds)[5].Name, link: "link", noCleanup: true},
		"d4": {real: (*cds)[6].Name, link: "link", noCleanup: false}}

	if diff := pretty.Compare(disks[w].m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateDisksValidate(t *testing.T) {
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d1"}}
	validatedImages = nameSet{w: {"i1"}}

	// Good cases.
	expectedType := func(p, z, t string) string { return fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", p, z, t) }
	defaultType := expectedType(w.Project, w.Zone, "pd-standard")
	goodTests := []struct {
		desc               string
		cd, wantCd         *CreateDisk
		wantValidatedDisks []string
	}{
		{
			"source image case",
			&CreateDisk{daisyName: "d2", Project: w.Project, Zone: w.Zone, Disk: compute.Disk{Name: "d2", SourceImage: "i1", Type: defaultType}},
			&CreateDisk{daisyName: "d2", Project: w.Project, Zone: w.Zone, Disk: compute.Disk{Name: "d2", SourceImage: "i1", Type: defaultType}},
			[]string{"d1", "d2"},
		},
		{
			"blank disk case",
			&CreateDisk{daisyName: "d3", Project: w.Project, Zone: w.Zone, SizeGb: "50", Disk: compute.Disk{Name: "d3", SizeGb: 50, Type: defaultType}},
			&CreateDisk{daisyName: "d3", Project: w.Project, Zone: w.Zone, SizeGb: "50", Disk: compute.Disk{Name: "d3", SizeGb: 50, Type: defaultType}},
			[]string{"d1", "d2", "d3"},
		},
	}

	for _, tt := range goodTests {
		cds := CreateDisks{tt.cd}
		if err := cds.validate(ctx, s); err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if diff := pretty.Compare(tt.cd, tt.wantCd); diff != "" {
			t.Errorf("%s: validated Disk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if !reflect.DeepEqual(validatedDisks[w], tt.wantValidatedDisks) {
			t.Errorf("%s: got:(%v) != want(%v)", tt.desc, validatedDisks[w], tt.wantValidatedDisks)
		}
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   *CreateDisk
		err  string
	}{
		{
			"bad Name case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "good-project", Zone: "good-zone", SizeGb: "50", Disk: compute.Disk{Name: "badName!", SizeGb: 50, Type: defaultType}},
			"cannot create disk: invalid name: \"badName!\"",
		},
		{
			"bad Project case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "badProject!", Zone: "good-zone", SizeGb: "50", Disk: compute.Disk{Name: "good-name", SizeGb: 50, Type: defaultType}},
			"cannot create disk: invalid project: \"badProject!\"",
		},
		{
			"bad Zone case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "good-project", Zone: "badZone!", SizeGb: "50", Disk: compute.Disk{Name: "good-name", SizeGb: 50, Type: defaultType}},
			"cannot create disk: invalid zone: \"badZone!\"",
		},
		{
			"bad Type case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "good-project", Zone: "good-zone", SizeGb: "50", Disk: compute.Disk{Name: "good-name", SizeGb: 50, Type: "badType!"}},
			"cannot create disk: invalid disk type: \"badType!\"",
		},
		{
			"dupe disk name case",
			&CreateDisk{daisyName: "d1", Project: "good-project", Zone: "good-zone", SizeGb: "50", Disk: compute.Disk{Name: "good-name", SizeGb: 50, Type: defaultType}},
			fmt.Sprintf("error adding disk: workflow %q has duplicate references for %q", w.Name, "d1"),
		},
		{
			"no size/source image case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "good-project", Zone: "good-zone", Disk: compute.Disk{Name: "good-name", Type: defaultType}},
			"cannot create disk: SizeGb and SourceImage not set",
		},
		{
			"image DNE case",
			&CreateDisk{daisyName: "good-daisy-name", Project: "good-project", Zone: "good-zone", Disk: compute.Disk{Name: "good-name", SourceImage: "i-dne", Type: defaultType}},
			"cannot create disk: image not found: \"i-dne\"",
		},
	}

	for _, tt := range badTests {
		cds := CreateDisks{tt.cd}
		if err := cds.validate(ctx, s); err == nil {
			t.Errorf("%s: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%s: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := []string{"d1", "d2", "d3"}
	if !reflect.DeepEqual(validatedDisks[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedDisks[w], want)
	}
}
