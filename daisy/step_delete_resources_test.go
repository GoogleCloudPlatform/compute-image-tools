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
	"errors"
	"fmt"
	"testing"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestDeleteResourcesPopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("s")
	s.DeleteResources = &DeleteResources{
		Disks:         []string{"d", "zones/z/disks/d"},
		Images:        []string{"i", "global/images/i"},
		MachineImages: []string{"i", "global/machineImages/i"},
		Instances:     []string{"i", "zones/z/instances/i"},
		Networks:      []string{"n", "global/networks/n"},
		Firewalls:     []string{"n", "global/firewalls/n"},
	}

	if err := (s.DeleteResources).populate(context.Background(), s); err != nil {
		t.Error("err should be nil")
	}

	want := &DeleteResources{
		Disks:         []string{"d", fmt.Sprintf("projects/%s/zones/z/disks/d", w.Project)},
		Images:        []string{"i", fmt.Sprintf("projects/%s/global/images/i", w.Project)},
		MachineImages: []string{"i", fmt.Sprintf("projects/%s/global/machineImages/i", w.Project)},
		Instances:     []string{"i", fmt.Sprintf("projects/%s/zones/z/instances/i", w.Project)},
		Networks:      []string{"n", fmt.Sprintf("projects/%s/global/networks/n", w.Project)},
		Firewalls:     []string{"n", fmt.Sprintf("projects/%s/global/firewalls/n", w.Project)},
	}
	if diffRes := diff(s.DeleteResources, want, 0); diffRes != "" {
		t.Errorf("DeleteResources not populated as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestDeleteResourcesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	s, _ := w.NewStep("s")
	ins := []*Resource{{RealName: "in0", link: "link"}, {RealName: "in1", link: "link"}, {RealName: "in2", link: "link"}}
	ims := []*Resource{{RealName: "im0", link: "link"}, {RealName: "im1", link: "link"}}
	mis := []*Resource{{RealName: "mi0", link: "link"}, {RealName: "mi1", link: "link"}}
	ds := []*Resource{{RealName: "d0", link: "link"}, {RealName: "d1", link: "link"}}
	ns := []*Resource{{RealName: "n0", link: "link"}, {RealName: "n1", link: "link"}}
	fs := []*Resource{{RealName: "f0", link: "link"}, {RealName: "f1", link: "link"}}
	w.instances.m = map[string]*Resource{"in0": ins[0], "in1": ins[1], "in2": ins[2]}
	w.images.m = map[string]*Resource{"im0": ims[0], "im1": ims[1]}
	w.machineImages.m = map[string]*Resource{"mi0": mis[0], "mi1": mis[1]}
	w.disks.m = map[string]*Resource{"d0": ds[0], "d1": ds[1]}
	w.networks.m = map[string]*Resource{"n0": ns[0], "n1": ns[1]}
	w.firewallRules.m = map[string]*Resource{"f0": fs[0], "f1": fs[1]}

	dr := &DeleteResources{
		Instances:     []string{"in0"},
		Images:        []string{"im0"},
		MachineImages: []string{"mi0"},
		Disks:         []string{"d0"},
		Networks:      []string{"n0"},
		GCSPaths:      []string{"gs://foo/bar"},
		Firewalls:     []string{"f0"},
	}
	if err := dr.run(ctx, s); err != nil {
		t.Fatalf("error running DeleteResources.run(): %v", err)
	}

	deletedChecks := []struct {
		r               *Resource
		shouldBeDeleted bool
	}{
		{ins[0], true},
		{ins[1], false},
		{ins[2], false},
		{ims[0], true},
		{ims[1], false},
		{mis[0], true},
		{mis[1], false},
		{ds[0], true},
		{ds[1], false},
		{ns[0], true},
		{ns[1], false},
		{fs[0], true},
		{fs[1], false},
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

	// Bad case
	if err := (&DeleteResources{GCSPaths: []string{"bkt"}}).run(ctx, s); err == nil {
		t.Error("run should have failed with a parsing error")
	}

	// Robustness check, no instance created but registered. Should retry a few times
	w.ComputeClient.(*daisyCompute.TestClient).GetInstanceFn = func(project, zone, name string) (*compute.Instance, error) {
		return nil, errors.New("foo")
	}
	retries := 0
	SleepFn = func(time.Duration) { retries++ }
	if err := (&DeleteResources{Instances: []string{"in2"}}).run(ctx, s); err != nil {
		t.Error("instance delete fail unexpectly")
	}
	if retries == 0 {
		t.Errorf("faulty instance delete didn't retry as expected. It run %v times", retries)
	}
}

func TestRecursiveGCSDelete(t *testing.T) {
	w := testWorkflow()
	ctx := context.Background()

	// Good case
	if err := recursiveGCSDelete(ctx, w, "foo", "bar"); err != nil {
		t.Error(err)
	}

	// Bad case
	if err := recursiveGCSDelete(ctx, w, "dne", "bar"); err == nil {
		t.Errorf("expected DNE error")
	}
}

func TestDeleteResourcesValidate(t *testing.T) {
	// Test:
	// - delete d0, im0, mi0 and in0 explicitly.
	// - make d1 an attached disk to in0 that has autoDelete = true.
	// - check that d0, d1, im0, in0 get registered for deletion.
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	w.cloudLoggingClient = nil
	dC, _ := w.NewStep("dCreator")
	imC, _ := w.NewStep("imCreator")
	miC, _ := w.NewStep("miCreator")
	inC, _ := w.NewStep("inCreator")
	nC, _ := w.NewStep("nCreator")
	fC, _ := w.NewStep("fCreator")
	s, _ := w.NewStep("s")
	w.AddDependency(s, dC, imC, miC, inC, nC, fC)
	otherDeleter, _ := w.NewStep("otherDeleter")
	ds := []*Resource{{RealName: "d0", link: "link", creator: dC}, {RealName: "d1", link: "link", creator: dC}, {RealName: "d2", link: "link", creator: dC}}
	ims := []*Resource{{RealName: "im0", link: "link", creator: imC}, {RealName: "im1", link: "link", creator: imC}}
	mis := []*Resource{{RealName: "mi0", link: "link", creator: miC}, {RealName: "mi1", link: "link", creator: miC}}
	ins := []*Resource{{RealName: "in0", link: "link", creator: inC}, {RealName: "in1", link: "link", creator: inC}}
	ns := []*Resource{{RealName: "n0", link: "link", creator: nC}, {RealName: "n1", link: "link", creator: nC}, {RealName: "n2", link: "link", creator: nC}}
	fs := []*Resource{{RealName: "f0", link: "link", creator: fC}, {RealName: "f1", link: "link", creator: fC}, {RealName: "f2", link: "link", creator: fC}}
	w.instances.m = map[string]*Resource{"in0": ins[0], "in1": ins[1]}
	w.images.m = map[string]*Resource{"im0": ims[0], "im1": ims[1]}
	w.machineImages.m = map[string]*Resource{"mi0": mis[0], "mi1": mis[1]}
	w.disks.m = map[string]*Resource{"d0": ds[0], "d1": ds[1]}
	w.networks.m = map[string]*Resource{"n0": ns[0], "n1": ns[1]}
	w.firewallRules.m = map[string]*Resource{"f0": fs[0], "f1": fs[1]}
	ads := []*compute.AttachedDisk{{Source: "d1"}}
	inC.CreateInstances = &CreateInstances{
		Instances: []*Instance{
			{
				InstanceBase: InstanceBase{
					Resource: Resource{daisyName: "in0"},
				},
				Instance: compute.Instance{Disks: ads},
			},
		},
	}

	CompareResources := func(got, want []*Resource) {
		for _, s := range []*Step{dC, imC, miC, inC, s, otherDeleter} {
			s.w = nil
		}
		if diffRes := diff(got, want, 0); diffRes != "" {
			t.Errorf("resources weren't registered for deletion as expected: (-got,+want)\n%s", diffRes)
		}
		for _, s := range []*Step{dC, imC, miC, inC, s, otherDeleter} {
			s.w = w
		}
	}

	// Good case.
	dr := DeleteResources{
		Disks:         []string{"d0"},
		Images:        []string{"im0", "projects/foo/global/images/" + testImage, "projects/foo/global/images/family/foo"},
		MachineImages: []string{"mi0", "projects/test-project/global/machineImages/" + testMachineImage},
		Instances:     []string{"in0"},
		Networks:      []string{"n0"},
		GCSPaths:      []string{"gs://foo/bar"},
		Firewalls:     []string{"f0"},
	}
	if err := dr.validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}
	got := []*Resource{ds[0], ds[1], ims[0], ims[1], mis[0], mis[1], ins[0], ins[1], ns[0], ns[1], fs[0], fs[1]}
	want := []*Resource{&(*ds[0]), &(*ds[1]), &(*ims[0]), &(*ims[1]), &(*mis[0]), &(*mis[1]), &(*ins[0]), &(*ins[1]), &(*ns[0]), &(*ns[1]), &(*fs[0]), &(*fs[1])}
	want[0].deleter = s
	want[1].deleter = s
	want[2].deleter = s
	want[5].deleter = s
	want[6].deleter = s
	want[8].deleter = s

	CompareResources(got, want)
	// Bad cases. Test:
	// - deleting an already deleted disk/image/instance/machine image (d1 is already deleted from other tests)
	// - deleting a disk that DNE
	ims[1].deleter = otherDeleter
	mis[1].deleter = otherDeleter
	ins[1].deleter = otherDeleter
	if err := (&DeleteResources{Disks: []string{"d1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted disk")
	}
	if err := (&DeleteResources{Images: []string{"im1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted image")
	}
	if err := (&DeleteResources{MachineImages: []string{"mi1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted machine image")
	}
	if err := (&DeleteResources{Instances: []string{"in1"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an already deleted instance")
	}
	if err := (&DeleteResources{Disks: []string{"dne"}}).validate(ctx, s); err == nil {
		t.Error("DeleteResources should have returned an error when deleting an disk that DNE")
	}
	if err := (&DeleteResources{Instances: []string{fmt.Sprintf("projects/%s/zones/%s/instances/dne", testProject, testZone)}}).validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}

	if err := (&DeleteResources{GCSPaths: []string{"dne"}}).validate(ctx, s); err == nil {
		t.Error("validation have failed with a parsing error")
	}
	if err := (&DeleteResources{GCSPaths: []string{"gs://dne"}}).validate(ctx, s); err == nil {
		t.Error("validation have failed with a DNE error")
	}

	want[3].deleter = otherDeleter
	want[5].deleter = otherDeleter
	CompareResources(got, want)
}
