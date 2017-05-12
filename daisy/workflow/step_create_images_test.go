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

func TestCreateImagesRun(t *testing.T) {
	w := testWorkflow()
	w.diskRefs.m = map[string]*resource{"d": {"d", w.genName("d"), "link", false}}
	w.Sources = map[string]string{"file": "gs://some/path"}
	ci := &CreateImages{
		{Name: "i1", SourceDisk: "d"},
		{Name: "i2", SourceFile: "gs://bucket/object"},
		{Name: "i2", SourceFile: "file", Project: "project"},
		{Name: "i3", SourceDisk: "d", NoCleanup: true},
		{Name: "i4", SourceDisk: "d", ExactName: true},
		{Name: "i5", SourceDisk: "zones/zone/disks/disk"},
	}
	if err := ci.run(w); err != nil {
		t.Errorf("error running CreateImages.run(): %v", err)
	}

	// Bad cases.
	badTests := []struct {
		name string
		cd   CreateImages
		err  string
	}{
		{
			"disk DNE",
			CreateImages{CreateImage{Name: "i-gaz", SourceDisk: "dne-disk"}},
			"unresolved instance reference \"dne-disk\"",
		},
		{
			"no disk/file",
			CreateImages{CreateImage{Name: "i-gaz"}},
			"you must provide either a sourceDisk or a sourceFile but not both to create an image",
		},
		{
			"bad GCS path",
			CreateImages{CreateImage{Name: "i-baz", SourceFile: "path/to/file"}},
			"\"path/to/file\" is not in Sources and is not a valid GCS path",
		},
	}

	for _, tt := range badTests {
		if err := tt.cd.run(w); err == nil {
			t.Errorf("%q: expected error, got nil", tt.name)
		} else if err.Error() != tt.err {
			t.Errorf("%q: did not get expected error from validate():\ngot: %q\nwant: %q", tt.name, err.Error(), tt.err)
		}
	}

	want := map[string]*resource{
		"i1": {"i1", w.genName("i1"), "link", false},
		"i2": {"i2", w.genName("i2"), "link", false},
		"i3": {"i3", w.genName("i3"), "link", true},
		"i4": {"i4", "i4", "link", false},
		"i5": {"i5", w.genName("i5"), "link", false},
	}

	if diff := pretty.Compare(w.imageRefs.m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
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
			CreateImages{{Name: "i-bar", SourceDisk: "d-foo"}},
			[]string{"i-foo", "i-bar"},
		},
		{
			"using sources file",
			CreateImages{{Name: "i-bas", SourceFile: "file"}},
			[]string{"i-foo", "i-bar", "i-bas"},
		},
		{
			"using GCS file",
			CreateImages{{Name: "i-baz", SourceFile: "gs://path/to/file"}},
			[]string{"i-foo", "i-bar", "i-bas", "i-baz"},
		},
	}

	for _, tt := range goodTests {
		if err := tt.cd.validate(w); err != nil {
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
			CreateImages{{Name: "i-baz", SourceFile: "gs://path/to/file"}},
			"error adding image: workflow \"\" has duplicate references for \"i-baz\"",
		},
		{
			"disk DNE",
			CreateImages{{Name: "i-gaz", SourceDisk: "dne-disk"}},
			"cannot create image: disk not found: dne-disk",
		},
		{
			"no disk/file",
			CreateImages{{Name: "i-gaz"}},
			"must provide either Disk or File, exclusively",
		},
		{
			"using both disk/file",
			CreateImages{{Name: "i-gaz", SourceDisk: "d-foo", SourceFile: "gs://path/to/file"}},
			"must provide either Disk or File, exclusively",
		},
		{
			"bad GCS path",
			CreateImages{{Name: "i-baz", SourceFile: "path/to/file"}},
			"cannot create image: file not in sources or valid GCS path: path/to/file",
		},
	}

	for _, tt := range badTests {
		if err := tt.cd.validate(w); err == nil {
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
