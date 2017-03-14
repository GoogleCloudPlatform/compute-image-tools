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
)

func TestCreateImagesRun(t *testing.T) {
	wf := testWorkflow()
	wf.createdDisks = map[string]string{namer("somedisk", wf.Name, wf.suffix): "link"}
	ci := &CreateImages{
		{Name: "image1", SourceDisk: "somedisk"},
		{Name: "image2", SourceFile: "somefile"},
		{Name: "image3", SourceDisk: "somedisk"}}
	if err := ci.run(wf); err != nil {
		t.Fatalf("error running CreateImages.run(): %v", err)
	}

	want := map[string]string{
		"image1": "link",
		"image2": "link",
		"image3": "link"}
	if !reflect.DeepEqual(wf.createdImages, want) {
		t.Errorf("Workflow.createdImages does not match expectations, got: %+v, want: %+v", wf.createdImages, want)
	}
}

func TestCreateImagesValidate(t *testing.T) {
	// Set up.
	diskNames = nameSet{"d-foo"}
	imageNames = nameSet{"i-foo"}
	defer func() {
		// Clean up.
		diskNames = nameSet{}
		imageNames = nameSet{}
	}()

	// Good case. Using disk.
	ci := CreateImages{CreateImage{Name: "i-bar", SourceDisk: "d-foo"}}
	if err := ci.validate(); err != nil {
		t.Fatal("validation should not have failed")
	}
	if !reflect.DeepEqual(imageNames, nameSet{"i-foo", "i-bar"}) {
		t.Fatalf("%s != %s", imageNames, nameSet{"i-foo", "i-bar"})
	}

	// Good case. Using file.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "/path/to/file"}}
	if err := ci.validate(); err != nil {
		t.Fatal("validation should not have failed")
	}
	if !reflect.DeepEqual(imageNames, nameSet{"i-foo", "i-bar", "i-baz"}) {
		t.Fatalf("%s != %s", imageNames, nameSet{"i-foo", "i-bar", "i-baz"})
	}

	// Bad case. Dupe name.
	ci = CreateImages{CreateImage{Name: "i-baz", SourceFile: "/path/to/file"}}
	if err := ci.validate(); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(imageNames, nameSet{"i-foo", "i-bar", "i-baz"}) {
		t.Fatalf("%s != %s", imageNames, nameSet{"i-foo", "i-bar", "i-baz"})
	}

	// Bad case. No disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz"}}
	if err := ci.validate(); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(imageNames, nameSet{"i-foo", "i-bar", "i-baz"}) {
		t.Fatalf("%s != %s", imageNames, nameSet{"i-foo", "i-bar", "i-baz"})
	}

	// Bad case. Using both disk/file.
	ci = CreateImages{CreateImage{Name: "i-gaz", SourceDisk: "d-foo", SourceFile: "/path/to/file"}}
	if err := ci.validate(); err == nil {
		t.Fatal("validation should have failed")
	}
	if !reflect.DeepEqual(imageNames, nameSet{"i-foo", "i-bar", "i-baz"}) {
		t.Fatalf("%s != %s", imageNames, nameSet{"i-foo", "i-bar", "i-baz"})
	}
}
