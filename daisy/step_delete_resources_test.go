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

package daisy

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/api/compute/v1"
)

func TestDeleteResourcesPopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("s")
	s.DeleteResources = &DeleteResources{
		Disks:     []string{"d", "zones/z/disks/d"},
		Images:    []string{"i", "global/images/i"},
		Instances: []string{"i", "zones/z/instances/i"},
	}

	if err := (s.DeleteResources).populate(context.Background(), s); err != nil {
		t.Error("err should be nil")
	}

	want := &DeleteResources{
		Disks:     []string{"d", fmt.Sprintf("projects/%s/zones/z/disks/d", w.Project)},
		Images:    []string{"i", fmt.Sprintf("projects/%s/global/images/i", w.Project)},
		Instances: []string{"i", fmt.Sprintf("projects/%s/zones/z/instances/i", w.Project)},
	}
	if diffRes := diff(s.DeleteResources, want, 0); diffRes != "" {
		t.Errorf("DeleteResources not populated as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDeleteResourcesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	s, _ := w.NewStep("s")
	ins := []*Resource{{RealName: "in0", link: "link"}, {RealName: "in1", link: "link"}}
	ims := []*Resource{{RealName: "im0", link: "link"}, {RealName: "im1", link: "link"}}
	ds := []*Resource{{RealName: "d0", link: "link"}, {RealName: "d1", link: "link"}}
	w.instances.m = map[string]*Resource{"in0": ins[0], "in1": ins[1]}
	w.images.m = map[string]*Resource{"im0": ims[0], "im1": ims[1]}
	w.disks.m = map[string]*Resource{"d0": ds[0], "d1": ds[1]}

	dr := &DeleteResources{Instances: []string{"in0"}, Images: []string{"im0"}, Disks: []string{"d0"}}
	if err := dr.run(ctx, s); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	deletedChecks := []struct {
		r               *Resource
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
				t.Errorf("resource %q should have been deleted", c.r.RealName)
			}
		} else if c.r.deleted {
			t.Errorf("resource %q should not have been deleted", c.r.RealName)
		}
	}
}

func TestDeleteResourcesValidate(t *testing.T) {
	// Test:
	// - delete d0, im0, and in0 explicitly.
	// - make d1 an attached disk to in0 that has autoDelete = true.
	// - check that d0, d1, im0, in0 get registered for deletion.
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	dC, _ := w.NewStep("dCreator")
	imC, _ := w.NewStep("imCreator")
	inC, _ := w.NewStep("inCreator")
	s, _ := w.NewStep("s")
	w.AddDependency(s, dC, imC, inC)
	otherDeleter, _ := w.NewStep("otherDeleter")
	ds := []*Resource{{RealName: "d0", link: "link", creator: dC}, {RealName: "d1", link: "link", creator: dC}}
	ims := []*Resource{{RealName: "im0", link: "link", creator: imC}, {RealName: "im1", link: "link", creator: imC}}
	ins := []*Resource{{RealName: "in0", link: "link", creator: inC}, {RealName: "in1", link: "link", creator: inC}}
	w.instances.m = map[string]*Resource{"in0": ins[0], "in1": ins[1]}
	w.images.m = map[string]*Resource{"im0": ims[0], "im1": ims[1]}
	w.disks.m = map[string]*Resource{"d0": ds[0], "d1": ds[1]}
	ads := []*compute.AttachedDisk{{Source: "d1"}}
	inC.CreateInstances = &CreateInstances{{Resource: Resource{daisyName: "in0"}, Instance: compute.Instance{Disks: ads}}}

	CompareResources := func(got, want []*Resource) {
		for _, s := range []*Step{dC, imC, inC, s, otherDeleter} {
			s.w = nil
		}
		if diffRes := diff(got, want, 0); diffRes != "" {
			t.Errorf("resources weren't registered for deletion as expected: (-got,+want)\n%s", diffRes)
		}
		for _, s := range []*Step{dC, imC, inC, s, otherDeleter} {
			s.w = w
		}
	}

	// Good case.
	dr := DeleteResources{Disks: []string{"d0"}, Images: []string{"im0"}, Instances: []string{"in0"}}
	if err := dr.validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	got := []*Resource{ds[0], ds[1], ims[0], ims[1], ins[0], ins[1]}
	want := []*Resource{&(*ds[0]), &(*ds[1]), &(*ims[0]), &(*ims[1]), &(*ins[0]), &(*ins[1])}
	want[0].deleter = s
	want[1].deleter = s
	want[2].deleter = s
	want[4].deleter = s

	CompareResources(got, want)
	// Bad cases. Test:
	// - deleting an already deleted disk/image/instance (d1 is already deleted from other tests)
	// - deleting a disk that DNE
	ims[1].deleter = otherDeleter
	ins[1].deleter = otherDeleter
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
	if err := (&DeleteResources{Instances: []string{fmt.Sprintf("projects/%s/zones/%s/instances/dne", testProject, testZone)}}).validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}

	want[3].deleter = otherDeleter
	want[5].deleter = otherDeleter
	CompareResources(got, want)
}
