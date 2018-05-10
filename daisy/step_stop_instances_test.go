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

func TestStopInstancesPopulate(t *testing.T) {
	w := testWorkflow()
	s, _ := w.NewStep("s")
	s.StopInstances = &StopInstances{
		Instances: []string{"i", "zones/z/instances/i"},
	}

	if err := (s.StopInstances).populate(context.Background(), s); err != nil {
		t.Error("err should be nil")
	}

	want := &StopInstances{
		Instances: []string{"i", fmt.Sprintf("projects/%s/zones/z/instances/i", w.Project)},
	}
	if diffRes := diff(s.StopInstances, want, 0); diffRes != "" {
		t.Errorf("StopInstances not populated as expected: (-got,+want)\n%s", diffRes)
	}
}

func TestStopInstanceValidate(t *testing.T) {
	ctx := context.Background()
	// Set up.
	w := testWorkflow()
	s, _ := w.NewStep("s")
	iCreator, _ := w.NewStep("iCreator")
	iCreator.CreateInstances = &CreateInstances{&Instance{}}
	w.AddDependency(s, iCreator)
	if err := w.instances.regCreate("instance1", &Resource{link: fmt.Sprintf("projects/%s/zones/%s/disks/d", testProject, testZone)}, iCreator); err != nil {
		t.Fatal(err)
	}

	if err := (&StopInstances{Instances: []string{"dne"}}).validate(ctx, s); err == nil {
		t.Error("StopInstances should have returned an error when stopping an instance that DNE")
	}
}
