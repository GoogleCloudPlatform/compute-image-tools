//  Copyright 2018 Google Inc. All Rights Reserved.
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
)

func TestStartInstancesPopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("s")
	s.StartInstances = &StartInstances{
		Instances: []string{"i", "zones/z/instances/i"},
	}

	if err := (s.StartInstances).populate(context.Background(), s); err != nil {
		t.Error("err should be nil")
	}

	want := &StartInstances{
		Instances: []string{"i", fmt.Sprintf("projects/%s/zones/z/instances/i", w.Project)},
	}
	if diffRes := diff(s.StartInstances, want, 0); diffRes != "" {
		t.Errorf("StartInstances not populated as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestStartInstancesValidate(t *testing.T) {
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	s, _ := w.NewStep("s")
	iCreator, _ := w.NewStep("iCreator")
	iCreator.CreateInstances = &CreateInstances{Instances: []*Instance{&Instance{}}}
	w.AddDependency(s, iCreator)
	if err := w.instances.regCreate("instance1", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}, false, iCreator); err != nil {
		t.Fatal(err)
	}

	if err := (&StartInstances{Instances: []string{"instance1"}}).validate(ctx, s); err != nil {
		t.Errorf("validation should not have failed: %v", err)
	}

	if err := (&StartInstances{Instances: []string{"dne"}}).validate(ctx, s); err == nil {
		t.Error("StartInstances should have returned an error when starting an instance that DNE")
	}
}

func TestStartInstancesRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()

	s, _ := w.NewStep("s")
	ins := []*Resource{{RealName: "in0", link: "link"}, {RealName: "in1", link: "link"}, {RealName: "in2", link: "link"}}
	w.instances.m = map[string]*Resource{"in0": ins[0], "in1": ins[1], "in2": ins[2]}

	stopI := &StopInstances{
		Instances: []string{"in1", "in2"},
	}
	if err := stopI.run(ctx, s); err != nil {
		t.Fatalf("error running StopInstances.run(): %v", err)
	}

	startI := &StartInstances{
		Instances: []string{"in1"},
	}
	if err := startI.run(ctx, s); err != nil {
		t.Fatalf("error running StartInstances.run(): %v", err)
	}

	startedChecks := []struct {
		r               *Resource
		shouldBeStarted bool
	}{
		{ins[0], true},
		{ins[1], true},
		{ins[2], false},
	}
	for _, c := range startedChecks {
		if c.shouldBeStarted {
			if c.r.stoppedByWf {
				t.Errorf("resource %q should have been started", c.r.RealName)
			}
		} else if !c.r.stoppedByWf {
			t.Errorf("resource %q should not have been started", c.r.RealName)
		}
	}
}
