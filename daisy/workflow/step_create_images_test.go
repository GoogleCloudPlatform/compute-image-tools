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

	"fmt"
	"github.com/kylelemons/godebug/pretty"
	compute "google.golang.org/api/compute/v1"
)

func TestCreateImagesRun(t *testing.T) {
	w := testWorkflow()
	var fakeClientErr error
	w.ComputeClient.CreateImageFake = func(project string, i *compute.Image) error {
		if fakeClientErr != nil {
			return fakeClientErr
		}
		i.SelfLink = "link"
		return nil
	}
	s := &Step{w: w}
	disks[w].m = map[string]*resource{"d": {"d", w.genName("d"), "link", false, false}}
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
			CreateImages{CreateImage{Image: compute.Image{Name: "i-gaz", SourceDisk: "dne-disk"}}},
			nil,
			"invalid or missing reference to SourceDisk \"dne-disk\"",
		},
		{
			"client failure",
			CreateImages{CreateImage{}},
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
		"i1": {"i1", (*cis)[0].Name, "link", false, false},
		"i2": {"i2", (*cis)[2].Name, "link", false, false},
		"i3": {"i3", (*cis)[3].Name, "link", true, false},
		"i4": {"i4", (*cis)[4].Name, "link", false, false},
		"i5": {"i5", (*cis)[5].Name, "link", false, false},
	}

	if diff := pretty.Compare(images[w].m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	// Set up.
	w := testWorkflow()
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"d-foo"}}
	validatedImages = nameSet{w: {"i-foo"}}
	w.Sources = map[string]string{"file": "gs://some/file"}

	// Good cases.
	goodTests := []struct {
		name string
		cd   CreateImages
		want []string
	}{
		{
			"using disk",
			CreateImages{{Image: compute.Image{Name: "i-bar", SourceDisk: "d-foo"}}},
			[]string{"i-foo", "i-bar"},
		},
		{
			"using sources file",
			CreateImages{{Image: compute.Image{Name: "i-bas", RawDisk: &compute.ImageRawDisk{Source: "file"}}}},
			[]string{"i-foo", "i-bar", "i-bas"},
		},
		{
			"using GCS file",
			CreateImages{{Image: compute.Image{Name: "i-baz", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}}}},
			[]string{"i-foo", "i-bar", "i-bas", "i-baz"},
		},
	}

	for _, tt := range goodTests {
		if err := tt.cd.validate(s); err != nil {
			t.Errorf("%q: unexpected error: %v", tt.name, err)
		}
		if !reflect.DeepEqual(validatedImages[w], tt.want) {
			t.Errorf("%q: got:(%v) != want(%v)", tt.name, validatedImages[w], tt.want)
		}
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   CreateImages
		err  string
	}{
		{
			"dupe name",
			CreateImages{{Image: compute.Image{Name: "i-baz", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}}}},
			fmt.Sprintf("error adding image: workflow %q has duplicate references for %q", w.Name, "i-baz"),
		},
		{
			"disk DNE",
			CreateImages{{Image: compute.Image{Name: "i-gaz", SourceDisk: "dne-disk"}}},
			"cannot create image: disk not found: dne-disk",
		},
		{
			"no disk/file",
			CreateImages{{Image: compute.Image{Name: "i-gaz"}}},
			"must provide either SourceDisk or RawDisk, exclusively",
		},
		{
			"using both disk/file",
			CreateImages{{Image: compute.Image{Name: "i-gaz", SourceDisk: "d-foo", RawDisk: &compute.ImageRawDisk{Source: "gs://path/to/file"}}}},
			"must provide either SourceDisk or RawDisk, exclusively",
		},
		{
			"bad GCS path",
			CreateImages{{Image: compute.Image{Name: "i-baz", RawDisk: &compute.ImageRawDisk{Source: "path/to/file"}}}},
			"cannot create image: file not in sources or valid GCS path: path/to/file",
		},
	}

	for _, tt := range badTests {
		if err := tt.cd.validate(s); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := []string{"i-foo", "i-bar", "i-bas", "i-baz"}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}
}
