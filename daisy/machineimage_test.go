//  Copyright 2020 Google Inc. All Rights Reserved.
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
	"strconv"
	"testing"

	"google.golang.org/api/compute/v1"
)

func TestMachineImagePopulate(t *testing.T) {
	w := testWorkflow()

	tests := []struct {
		desc      string
		mi        *MachineImage
		shouldErr bool
	}{
		{"default case", &MachineImage{}, false},
	}

	for testNum, tt := range tests {
		s, _ := w.NewStep("s" + strconv.Itoa(testNum))
		err := tt.mi.populate(context.Background(), s)

		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}

func TestMachineImagesValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.instances.m = map[string]*Resource{
		"si": {link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", w.Project, w.Zone, "si")},
	}
	s, e1 := w.NewStep("s")
	var e2 error
	w.ComputeClient, e2 = newTestGCEClient()
	if errs := addErrs(nil, e1, e2); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	tests := []struct {
		desc      string
		mi        *MachineImage
		shouldErr bool
	}{
		{"simple case success", &MachineImage{MachineImage: compute.MachineImage{Name: "i1", SourceInstance: "si"}}, false},
		{"no source instance case failure", &MachineImage{MachineImage: compute.MachineImage{Name: "i2"}}, true},
	}

	for _, tt := range tests {
		s.CreateMachineImages = &CreateMachineImages{tt.mi}

		// Test sanitation -- clean/set irrelevant fields.
		tt.mi.daisyName = tt.mi.Name
		tt.mi.RealName = tt.mi.Name
		tt.mi.link = fmt.Sprintf("projects/%s/global/machineImages/%s", w.Project, tt.mi.Name)
		tt.mi.Project = w.Project // Resource{} fields are tested in resource_test.

		if err := s.validate(ctx); tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
