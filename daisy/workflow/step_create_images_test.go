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
	wf.diskRefs.m = map[string]*Resource{"d": {"d", wf.ephemeralName("d"), "link", false}}
	ci := &CreateImages{
		{Name: "i1", SourceDisk: "d"},
		{Name: "i2", SourceFile: "f"},
		{Name: "i3", SourceDisk: "d"},
	}
	if err := ci.run(wf); err != nil {
		t.Fatalf("error running CreateImages.run(): %v", err)
	}

	want := map[string]*Resource{
		"i1": {"i1", wf.ephemeralName("i1"), "link", false},
		"i2": {"i2", wf.ephemeralName("i2"), "link", false},
		"i3": {"i3", wf.ephemeralName("i3"), "link", false},
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
		t.Fatal("validation should not have failed")
	}
	if !reflect.DeepEqual(validatedImages, nameSet{w: {"i-foo", "i-bar"}}) {
		t.Fatalf("%v != %v", validatedImages, nameSet{w: {"i-foo", "i-bar"}})
	}

	// Good case. Using file.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "/path/to/file"}}
	if err := ci.validate(w); err != nil {
		t.Fatal("validation should not have failed")
	}
	if !reflect.DeepEqual(validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}}) {
		t.Fatalf("%v != %v", validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}})
	}

	// Bad case. Dupe name.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "/path/to/file"}}
	if err := ci.validate(w); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}}) {
		t.Fatalf("%v != %v", validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}})
	}

	// Bad case. No disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz"}}
	if err := ci.validate(w); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}}) {
		t.Fatalf("%v != %v", validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}})
	}

	// Bad case. Using both disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz", SourceDisk: "d-foo", SourceFile: "/path/to/file"}}
	if err := ci.validate(w); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}}) {
		t.Fatalf("%v != %v", validatedImages, nameSet{w: {"i-foo", "i-bar", "i-baz"}})
	}
}
