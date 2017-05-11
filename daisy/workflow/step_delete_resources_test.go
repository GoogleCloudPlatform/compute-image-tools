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
	"testing"

	"reflect"

	"github.com/kylelemons/godebug/pretty"
)

func TestDeleteResourcesRun(t *testing.T) {
	wf := testWorkflow()
	wf.instanceRefs.m = map[string]*resource{
		"in1": {"in1", wf.genName("in1"), "link", false},
		"in2": {"in2", wf.genName("in2"), "link", false},
		"in3": {"in3", wf.genName("in3"), "link", false},
		"in4": {"in4", wf.genName("in4"), "link", false}}
	wf.imageRefs.m = map[string]*resource{
		"im1": {"im1", wf.genName("im1"), "link", false},
		"im2": {"im2", wf.genName("im2"), "link", false},
		"im3": {"im3", wf.genName("im3"), "link", false},
		"im4": {"im4", wf.genName("im4"), "link", false}}
	wf.diskRefs.m = map[string]*resource{
		"d1": {"d1", wf.genName("d1"), "link", false},
		"d2": {"d2", wf.genName("d2"), "link", false},
		"d3": {"d3", wf.genName("d3"), "link", false},
		"d4": {"d4", wf.genName("d4"), "link", false}}

	dr := &DeleteResources{
		Instances: []string{"in1", "in2", "in3"},
		Images:    []string{"im1", "im2", "im3"},
		Disks:     []string{"d1", "d2", "d3"}}
	if err := dr.run(wf); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	want := map[string]*resource{"in4": {"in4", wf.genName("in4"), "link", false}}
	if diff := pretty.Compare(wf.instanceRefs.m, want); diff != "" {
		t.Errorf("instanceRefs do not match expectation: (-got +want)\n%s", diff)
	}

	want = map[string]*resource{"im4": {"im4", wf.genName("im4"), "link", false}}
	if diff := pretty.Compare(wf.imageRefs.m, want); diff != "" {
		t.Errorf("imageRefs do not match expectation: (-got +want)\n%s", diff)
	}

	want = map[string]*resource{"d4": {"d4", wf.genName("d4"), "link", false}}
	if diff := pretty.Compare(wf.diskRefs.m, want); diff != "" {
		t.Errorf("diskRefs do not match expectation: (-got +want)\n%s", diff)
	}

	wf = testWorkflow()
	dr = &DeleteResources{
		Disks: []string{"notexist"}}
	close(wf.Cancel)
	if err := dr.run(wf); err != nil {
		t.Errorf("Should not error on non existent disk when Cancel is closed: %v", err)
	}

	// Bad cases.
	wf = testWorkflow()
	tests := []struct {
		dr  DeleteResources
		err string
	}{
		{
			DeleteResources{Disks: []string{"notexist"}},
			"unresolved disk \"notexist\"",
		},
		{
			DeleteResources{Instances: []string{"notexist"}},
			"unresolved instance \"notexist\"",
		},
		{
			DeleteResources{Images: []string{"notexist"}},
			"unresolved image \"notexist\"",
		},
	}

	for _, tt := range tests {
		if err := tt.dr.run(wf); err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != tt.err {
			t.Errorf("did not get expected error from validate():\ngot: %q\nwant: %q", err.Error(), tt.err)
		}
	}
}

func TestDeleteResourcesValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	validatedDisks = nameSet{w: {"foo", ":/#"}}
	validatedInstances = nameSet{w: {"foo", ":/#"}}
	validatedImages = nameSet{w: {"foo", ":/#"}}

	// Good case.
	dr := DeleteResources{
		Instances: []string{"foo"}, Disks: []string{"foo"}, Images: []string{"foo"},
	}
	if err := dr.validate(w); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}

	// Bad cases.
	tests := []struct {
		dr  DeleteResources
		err string
	}{
		{
			DeleteResources{Disks: []string{":/#"}},
			"error scheduling disk for deletion: bad name \":/#\"",
		},
		{
			DeleteResources{Instances: []string{":/#"}},
			"error scheduling instance for deletion: bad name \":/#\"",
		},
		{
			DeleteResources{Images: []string{":/#"}},
			"error scheduling image for deletion: bad name \":/#\"",
		},
		{
			DeleteResources{Disks: []string{"baz"}},
			"cannot delete disk, disk not found: baz",
		},
		{
			DeleteResources{Instances: []string{"baz"}},
			"cannot delete instance, instance not found: baz",
		},
		{
			DeleteResources{Images: []string{"baz"}},
			"cannot delete image, image not found: baz",
		},
	}

	for _, tt := range tests {
		if err := tt.dr.validate(w); err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != tt.err {
			t.Errorf("did not get expected error from validate():\ngot: %q\nwant: %q", err.Error(), tt.err)
		}
	}

	want := []string{"foo"}
	if !reflect.DeepEqual(validatedDiskDeletions[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedDisks[w], want)
	}

	if !reflect.DeepEqual(validatedInstanceDeletions[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedInstances[w], want)
	}

	if !reflect.DeepEqual(validatedImageDeletions[w], want) {
		t.Errorf("got:(%v) != want(%v)", validatedImages[w], want)
	}
}
