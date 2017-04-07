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
	wf := testWorkflow()
	wf.diskRefs.m = map[string]*resource{"d": {"d", wf.genName("d"), "link", false}}
	wf.Sources = map[string]string{"file": "gs://some/path"}
	ci := &CreateImages{
		{Name: "i1", SourceDisk: "d"},
		{Name: "i2", SourceFile: "gs://bucket/object"},
		{Name: "i2", SourceFile: "file"},
		{Name: "i3", SourceDisk: "d", NoCleanup: true},
		{Name: "i4", SourceDisk: "d", ExactName: true},
	}
	if err := ci.run(wf); err != nil {
		t.Fatalf("error running CreateImages.run(): %v", err)
	}

	want := map[string]*resource{
		"i1": {"i1", wf.genName("i1"), "link", false},
		"i2": {"i2", wf.genName("i2"), "link", false},
		"i3": {"i3", wf.genName("i3"), "link", true},
		"i4": {"i4", "i4", "link", false},
	}

	if diff := pretty.Compare(wf.imageRefs.m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedDisks = nameSet{w: {"d-foo"}}
	validatedImages = nameSet{w: {"i-foo"}}

	// Good case. Using disk.
	ci := CreateImages{CreateImage{Name: "i-bar", SourceDisk: "d-foo"}}
	if err := ci.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	want := []string{"i-foo", "i-bar"}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}

	// Good case. Using sources file.
	w.Sources = map[string]string{"file": "gs://some/file"}
	ci = CreateImages{CreateImage{Name: "i-bas", SourceFile: "file"}}
	if err := ci.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	want = []string{"i-foo", "i-bar", "i-bas"}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}

	// Good case. Using GCS file.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "gs://path/to/file"}}
	if err := ci.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	want = []string{"i-foo", "i-bar", "i-bas", "i-baz"}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}

	// Bad case. Dupe name.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "gs://path/to/file"}}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}

	// Bad case. No disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz"}}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}

	// Bad case. Using both disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz", SourceDisk: "d-foo", SourceFile: "gs://path/to/file"}}
	if err := ci.validate(w); err == nil {
		t.Errorf("validation should have failed: %v", err)
	}
	if !reflect.DeepEqual(validatedImages[w], want) {
		t.Fatalf("got:(%v) != want(%v)", validatedImages[w], want)
	}
}
