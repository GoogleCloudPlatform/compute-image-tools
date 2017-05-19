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
)

func TestDeleteResourcesRun(t *testing.T) {
	w := testWorkflow()
	s := &Step{w: w}
	ins := []*resource{
		{"in0", w.genName("in0"), "link", false, false},
		{"in1", w.genName("in1"), "link", false, false},
		{"in2", w.genName("in2"), "link", false, false},
		{"in3", w.genName("in3"), "link", false, false},
	}
	ims := []*resource{
		{"im0", w.genName("im0"), "link", false, false},
		{"im1", w.genName("im1"), "link", false, false},
		{"im2", w.genName("im2"), "link", false, false},
		{"im3", w.genName("im3"), "link", false, false},
	}
	ds := []*resource{
		{"d0", w.genName("d0"), "link", false, false},
		{"d1", w.genName("d1"), "link", false, false},
		{"d2", w.genName("d2"), "link", false, false},
		{"d3", w.genName("d3"), "link", false, false},
	}
	instances[w].m = map[string]*resource{"in0": ins[0], "in1": ins[1], "in2": ins[2], "in3": ins[3]}
	images[w].m = map[string]*resource{"im0": ims[0], "im1": ims[1], "im2": ims[2], "im3": ims[3]}
	disks[w].m = map[string]*resource{"d0": ds[0], "d1": ds[1], "d2": ds[2], "d3": ds[3]}

	dr := &DeleteResources{
		Instances: []string{"in0", "in1", "in2"},
		Images:    []string{"im0", "im1", "im2"},
		Disks:     []string{"d0", "d1", "d2"}}
	if err := dr.run(s); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	for _, rs := range [][]*resource{ins, ims, ds} {
		for i := 0; i < 3; i++ {
			if r := rs[i]; !r.deleted {
				t.Errorf("resource %q should have been deleted", r.name)
			}
		}
		if rs[3].deleted {
			t.Errorf("resource %q should not have been deleted", rs[3].name)
		}
	}

	w = testWorkflow()
	s = &Step{w: w}
	dr = &DeleteResources{
		Disks: []string{"notexist"}}
	close(w.Cancel)
	if err := dr.run(s); err != nil {
		t.Errorf("Should not error on non existent disk when Cancel is closed: %v", err)
	}

	// Bad cases.
	w = testWorkflow()
	s = &Step{w: w}
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
		if err := tt.dr.run(s); err == nil {
			t.Error("expected error, got nil")
		} else if err.Error() != tt.err {
			t.Errorf("did not get expected error from validate():\ngot: %q\nwant: %q", err.Error(), tt.err)
		}
	}
}

func TestDeleteResourcesValidate(t *testing.T) {
	// Set up.
	w := &Workflow{}
	s := &Step{w: w}
	validatedDisks = nameSet{w: {"foo", ":/#"}}
	validatedInstances = nameSet{w: {"foo", ":/#"}}
	validatedImages = nameSet{w: {"foo", ":/#"}}

	// Good case.
	dr := DeleteResources{
		Instances: []string{"foo"}, Disks: []string{"foo"}, Images: []string{"foo"},
	}
	if err := dr.validate(s); err != nil {
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
		if err := tt.dr.validate(s); err == nil {
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
