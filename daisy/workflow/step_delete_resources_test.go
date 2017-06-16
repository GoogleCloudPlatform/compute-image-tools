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
	ins := []*resource{{real: "in0", link: "link"}, {real: "in1", link: "link"}}
	ims := []*resource{{real: "im0", link: "link"}, {real: "im1", link: "link"}}
	ds := []*resource{{real: "d0", link: "link"}, {real: "d1", link: "link"}}
	instances[w].m = map[string]*resource{"in0": ins[0], "in1": ins[1]}
	images[w].m = map[string]*resource{"im0": ims[0], "im1": ims[1]}
	disks[w].m = map[string]*resource{"d0": ds[0], "d1": ds[1]}

	dr := &DeleteResources{Instances: []string{"in0"}, Images: []string{"im0"}, Disks: []string{"d0"}}
	if err := dr.run(s); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	deletedChecks := []struct {
		r               *resource
		shouldBeDeleted bool
	}{
		{ins[0], true},
		{ins[1], false},
		{ims[0], true},
		{ims[1], false},
		{ds[0], true},
		{ds[1], false},
	}
	for _, c := range deletedChecks {
		if c.shouldBeDeleted {
			if !c.r.deleted {
				t.Errorf("resource %q should have been deleted", c.r.real)
			}
			if c.r.deleter != s {
				t.Errorf("resource %q should have the deletion step as its deleter, but doesn't", c.r.real)
			}
		} else if c.r.deleted {
			t.Errorf("resource %q should not have been deleted", c.r.real)
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
