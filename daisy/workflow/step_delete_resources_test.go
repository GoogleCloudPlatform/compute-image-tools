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
	"testing"
)

func TestDeleteResourcesPopulate(t *testing.T) {
	if err := (&DeleteResources{}).populate(context.Background(), &Step{}); err != nil {
		t.Error("not implemented, err should be nil")
	}
}

func TestDeleteResourcesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	ins := []*resource{{real: "in0", link: "link"}, {real: "in1", link: "link"}}
	ims := []*resource{{real: "im0", link: "link"}, {real: "im1", link: "link"}}
	ds := []*resource{{real: "d0", link: "link"}, {real: "d1", link: "link"}}
	instances[w].m = map[string]*resource{"in0": ins[0], "in1": ins[1]}
	images[w].m = map[string]*resource{"im0": ims[0], "im1": ims[1]}
	disks[w].m = map[string]*resource{"d0": ds[0], "d1": ds[1]}

	dr := &DeleteResources{Instances: []string{"in0"}, Images: []string{"im0"}, Disks: []string{"d0"}}
	if err := dr.run(ctx, s); err != nil {
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
}

func TestDeleteResourcesValidate(t *testing.T) {
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	s := &Step{name: "s", w: w}
	dCreator := &Step{name: "dCreator", w: w}
	imCreator := &Step{name: "imCreator", w: w}
	inCreator := &Step{name: "inCreator", w: w}
	otherDeleter := &Step{}
	w.Steps = map[string]*Step{"s": s, "dCreator": dCreator, "imCreator": imCreator, "inCreator": inCreator}
	w.Dependencies = map[string][]string{"s": {"dCreator", "imCreator", "inCreator"}}
	ds := []*resource{{real: "d0", link: "link", creator: dCreator}, {real: "d1", link: "link", creator: dCreator}}
	ims := []*resource{{real: "im0", link: "link", creator: imCreator}, {real: "im1", link: "link", creator: imCreator}}
	ins := []*resource{{real: "in0", link: "link", creator: inCreator}, {real: "in1", link: "link", creator: inCreator}}
	instances[w].m = map[string]*resource{"in0": ins[0], "in1": ins[1]}
	images[w].m = map[string]*resource{"im0": ims[0], "im1": ims[1]}
	disks[w].m = map[string]*resource{"d0": ds[0], "d1": ds[1]}

	// Good case.
	dr := DeleteResources{Disks: []string{"d0"}, Images: []string{"im0"}, Instances: []string{"in0"}}
	if err := dr.validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	for _, l := range [][]*resource{ds, ims, ins} {
		if l[0].deleter != s {
			t.Errorf("Resource %q incorrect deleter: got: %v, want: %v", l[0].real, l[0].deleter, s)
		}
	}

	// Bad cases.
	// Test failure for each resource type (by marking each type as already registered for deletion).
	// Test deleting a resource that DNE.
	for _, l := range [][]*resource{ds, ims, ins} {
		l[1].deleter = otherDeleter
	}

	if err := (&DeleteResources{Disks: []string{"d1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted disk")
	}
	if err := (&DeleteResources{Images: []string{"im1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted image")
	}
	if err := (&DeleteResources{Instances: []string{"in1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted instance")
	}
	if err := (&DeleteResources{Disks: []string{"dne"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted disk")
	}
	for _, l := range [][]*resource{ds, ims, ins} {
		if l[1].deleter != otherDeleter {
			t.Errorf("Resource %q should not have changed deleters: got: %v, want: %v", l[1].real, l[1].deleter, otherDeleter)
		}
	}
}
