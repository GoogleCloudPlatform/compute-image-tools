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

func TestSnapshotPopulate(t *testing.T) {
	w := testWorkflow()

	currentTestProject := "currenttestproj"
	currentTestZone := "currenttestzone"
	currentTestDisk := "currenttestdisk"

	tests := []struct {
		desc                string
		ss                  *Snapshot
		shouldErr           bool
		expectedExtendedURL string
		expectedLink        string
	}{
		{"source disk URI: only name", &Snapshot{
			Resource: Resource{ExactName: true},
			Snapshot: compute.Snapshot{Name: testSnapshot, SourceDisk: fmt.Sprintf("aaa")}}, false,
			"aaa", fmt.Sprintf("projects/%s/global/snapshots/%s", testProject, testSnapshot),
		},
		{"source disk URI: with zones", &Snapshot{
			Resource: Resource{ExactName: true},
			Snapshot: compute.Snapshot{Name: testSnapshot, SourceDisk: fmt.Sprintf("zones/%v/disks/%v", currentTestZone, currentTestDisk)}}, false,
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", testProject, currentTestZone, currentTestDisk), fmt.Sprintf("projects/%s/global/snapshots/%s", testProject, testSnapshot),
		},
		{"source disk URI: with projects and zones", &Snapshot{
			Resource: Resource{ExactName: true},
			Snapshot: compute.Snapshot{Name: testSnapshot, SourceDisk: fmt.Sprintf("projects/%v/zones/%v/disks/%v", currentTestProject, currentTestZone, currentTestDisk)}}, false,
			fmt.Sprintf("projects/%v/zones/%v/disks/%v", currentTestProject, currentTestZone, currentTestDisk), fmt.Sprintf("projects/%s/global/snapshots/%s", testProject, testSnapshot),
		},
	}

	for testNum, tt := range tests {
		s, _ := w.NewStep("s" + strconv.Itoa(testNum))
		err := tt.ss.populate(context.Background(), s)

		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned error but didn't", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}

		if !tt.shouldErr && err == nil {
			if tt.expectedExtendedURL != tt.ss.SourceDisk {
				t.Errorf("%s: expected extended source disk url: '%v', actual: '%v'", tt.desc, tt.expectedExtendedURL, tt.ss.SourceDisk)
			}
			if tt.expectedLink != tt.ss.link {
				t.Errorf("%s: expected link: '%v', actual: '%v'", tt.desc, tt.expectedLink, tt.ss.link)
			}
		}
	}
}

func TestSnapshotValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.disks.m = map[string]*Resource{
		"sd": {link: fmt.Sprintf("projects/%s/zones/%s/disks/%s", w.Project, w.Zone, "sd")},
	}
	s, e1 := w.NewStep("s")
	var e2 error
	w.ComputeClient, e2 = newTestGCEClient()
	if errs := addErrs(nil, e1, e2); errs != nil {
		t.Fatalf("test set up error: %v", errs)
	}

	tests := []struct {
		desc      string
		ss        *Snapshot
		shouldErr bool
	}{
		{"no source disk case failure", &Snapshot{Snapshot: compute.Snapshot{Name: "ss1"}}, true},
		{"source disk created by daisy", &Snapshot{Snapshot: compute.Snapshot{Name: "ss2", SourceDisk: "sd"}}, false},
		{"source disk with disk size", &Snapshot{Snapshot: compute.Snapshot{Name: "ss3", SourceDisk: "sd", DiskSizeGb: 100}}, false},
		{"source disk URI: only name", &Snapshot{Snapshot: compute.Snapshot{Name: "ss4", SourceDisk: fmt.Sprintf("aaa")}}, true},
		{"source disk URI: with zones", &Snapshot{Snapshot: compute.Snapshot{Name: "ss5", SourceDisk: fmt.Sprintf("zones/%v/disks/%v", testZone, testDisk)}}, false},
		{"source disk URI: with projects and zones", &Snapshot{Snapshot: compute.Snapshot{Name: "ss6", SourceDisk: fmt.Sprintf("projects/%v/zones/%v/disks/%v", testProject, testZone, testDisk)}}, false},
	}

	for _, tt := range tests {
		s.CreateSnapshots = &CreateSnapshots{tt.ss}

		// Test sanitation -- clean/set irrelevant fields.
		tt.ss.daisyName = tt.ss.Name
		tt.ss.RealName = tt.ss.Name
		tt.ss.link = fmt.Sprintf("projects/%s/global/snapshots/%s", w.Project, tt.ss.Name)
		tt.ss.Project = w.Project // Resource{} fields are tested in resource_test.

		err := s.populate(ctx)
		if err == nil {
			err = s.validate(ctx)
		}

		if tt.shouldErr && err == nil {
			t.Errorf("%s: should have returned an error", tt.desc)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
	}
}
