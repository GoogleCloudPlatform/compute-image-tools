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

func TestCreateImagesRun(t *testing.T) {
	w := testWorkflow()
	s := &Step{w: w}
	disks[w].m = map[string]*resource{"d": {real: w.genName("d"), link: "link"}}
	w.Sources = map[string]string{"file": "gs://some/path"}
	cis := &CreateImages{
		{name: "i1", Image: compute.Image{Name: "i1", SourceDisk: "d"}},
		{name: "i2", Image: compute.Image{Name: "i2", RawDisk: &compute.ImageRawDisk{Source: "gs://bucket/object"}}},
		{name: "i2", Image: compute.Image{Name: "i2", RawDisk: &compute.ImageRawDisk{Source: "file"}}, Project: "project"},
		{name: "i3", Image: compute.Image{Name: "i3", SourceDisk: "d"}, NoCleanup: true},
		{name: "i4", Image: compute.Image{Name: "i4", SourceDisk: "d"}, ExactName: true},
		{name: "i5", Image: compute.Image{Name: "i5", SourceDisk: "zones/zone/disks/disk"}},
	}
	if err := cis.run(s); err != nil {
		t.Errorf("error running CreateImages.run(): %v", err)
	}

	// Bad cases.
	badTests := []struct {
		name          string
		cd            CreateImages
		fakeClientErr error
		err           string
	}{
		{
			"disk DNE",
			CreateImages{{Image: compute.Image{Name: "i-gaz", SourceDisk: "dne-disk"}}},
			nil,
			"invalid or missing reference to SourceDisk \"dne-disk\"",
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
		"i1": {real: (*cis)[0].Name, link: "link"},
		"i2": {real: (*cis)[2].Name, link: "link"},
		"i3": {real: (*cis)[3].Name, link: "link", noCleanup: true},
		"i4": {real: (*cis)[4].Name, link: "link"},
		"i5": {real: (*cis)[5].Name, link: "link"},
	}

	if diff := pretty.Compare(images[w].m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	// Set up.
	w := testWorkflow()
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d1"}}
	validatedImages = nameSet{w: {"i1"}}
	w.Sources = map[string]string{"file": "gs://some/file"}

	gcsAPIPath1 := w.getSourceGCSAPIPath("file")
	gcsAPIPath2, _ := getGCSAPIPath("gs://path/to/file")

	// Good cases.
	goodTests := []struct {
		desc                string
		ci, wantCi          *CreateImage
		wantValidatedImages []string
	}{
		{
			"using disk",
			&CreateImage{Image: compute.Image{Name: "i2", SourceDisk: "d1", Description: "foo"}},
			&CreateImage{name: "i2", Image: compute.Image{Name: w.genName("i2"), SourceDisk: "d1", Description: "foo"}, Project: w.Project},
			[]string{"i1", "i2"},
		},
		{
			"using sources file",
			&CreateImage{Image: compute.Image{Name: "i3", RawDisk: &compute.ImageRawDisk{Source: "file"}, Description: "foo"}},
			&CreateImage{name: "i3", Image: compute.Image{Name: w.genName("i3"), RawDisk: &compute.ImageRawDisk{Source: gcsAPIPath1}, Description: "foo"}, Project: w.Project},
			[]string{"i1", "i2", "i3"},
		},
		{
			"using GCS file",
			&CreateImage{Image: compute.Image{Name: "i4", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}, Description: "foo"}},
			&CreateImage{name: "i4", Image: compute.Image{Name: w.genName("i4"), RawDisk: &compute.ImageRawDisk{Source: gcsAPIPath2}, Description: "foo"}, Project: w.Project},
			[]string{"i1", "i2", "i3", "i4"},
		},
		{
			"exact name",
			&CreateImage{Image: compute.Image{Name: "i5", SourceDisk: "d1", Description: "foo"}, ExactName: true},
			&CreateImage{name: "i5", Image: compute.Image{Name: "i5", SourceDisk: "d1", Description: "foo"}, Project: w.Project, ExactName: true},
			[]string{"i1", "i2", "i3", "i4", "i5"},
		},
		{
			"non default project",
			&CreateImage{Image: compute.Image{Name: "i6", SourceDisk: "d1", Description: "foo"}, Project: "foo-project"},
			&CreateImage{name: "i6", Image: compute.Image{Name: w.genName("i6"), SourceDisk: "d1", Description: "foo"}, Project: "foo-project"},
			[]string{"i1", "i2", "i3", "i4", "i5", "i6"},
		},
	}

	for _, tt := range goodTests {
		cis := CreateImages{tt.ci}
		if err := cis.validate(s); err != nil {
			t.Errorf("%q: unexpected error: %v", tt.desc, err)
		}
		if diff := pretty.Compare(tt.ci, tt.wantCi); diff != "" {
			t.Errorf("%q: validated Disk does not match expectation: (-got +want)\n%s", tt.desc, diff)
		}
		if !reflect.DeepEqual(validatedImages[w], tt.wantValidatedImages) {
			t.Errorf("%q: got:(%v) != want(%v)", tt.desc, validatedImages[w], tt.wantValidatedImages)
		}
	}

	// Bad cases.
	badTests := []struct {
		name string
		ci   *CreateImage
		err  string
	}{
		{
			"dupe name",
			&CreateImage{Image: compute.Image{Name: "i1", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}}},
			fmt.Sprintf("error adding image: workflow %q has duplicate references for %q", w.Name, "i1"),
		},
		{
			"disk DNE",
			&CreateImage{Image: compute.Image{Name: "bi1", SourceDisk: "dne-disk"}},
			"cannot create image: disk not found: dne-disk",
		},
		{
			"no disk/file",
			&CreateImage{Image: compute.Image{Name: "bi1"}},
			"must provide either SourceDisk or RawDisk, exclusively",
		},
		{
			"using both disk/file",
			&CreateImage{Image: compute.Image{Name: "bi1", SourceDisk: "d-foo", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}}},
			"must provide either SourceDisk or RawDisk, exclusively",
		},
		{
			"bad GCS path",
			&CreateImage{Image: compute.Image{Name: "bi1", RawDisk: &compute.ImageRawDisk{Source: "path/to/file"}}},
			"cannot create image: file not in sources or valid GCS path: path/to/file",
		},
	}

	for _, tt := range badTests {
		cis := CreateImages{tt.ci}
		if err := cis.validate(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := []string{"i1", "i2", "i3", "i4", "i5", "i6"}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}
}
