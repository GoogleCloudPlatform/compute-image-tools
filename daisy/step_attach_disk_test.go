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

func TestAttachDisksPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	ads := &AttachDisks{{AttachedDisk: compute.AttachedDisk{Source: "someDisk"}}}
	if err := ads.populate(ctx, s); err != nil {
		t.Fatal(err)
	}
	want := AttachDisks{{AttachedDisk: compute.AttachedDisk{Mode: defaultDiskMode, DeviceName: "someDisk", Source: "someDisk"}}}
	if diffRes := diff(*ads, want, 0); diffRes != "" {
		t.Errorf(diffRes)
	}
}

func TestAttachDisksPopulateAndValidate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}
	w.disks.m = map[string]*Resource{
		testDisk: {Project: testProject, RealName: testDisk, link: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)},
		"bad":    {Project: "bad", RealName: testDisk, link: "link"},
	}

	tests := []struct {
		desc    string
		ads     *AttachDisks
		wantErr bool
	}{
		{"bad mode case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: "bad", Source: testDisk}}}, true},
		{"empty source case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW}}}, true},
		{"bad source case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: "bad"}}}, true},
		{"bad instance case", &AttachDisks{{Instance: "bad", AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: testDisk}}}, true},
		{"bad project case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: fmt.Sprintf("projects/bad/zones/%s/disks/bad", testZone)}}}, true},
		{"bad zone case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: fmt.Sprintf("projects/%s/zones/bad/disks/bad", testProject)}}}, true},
		{"url case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}}, false},
		{"resolve instance and disk case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: testDisk}}}, false},
	}
	for _, tt := range tests {
		var err error
		if err = tt.ads.populate(ctx, s); err == nil {
			err = tt.ads.validate(ctx, s)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}

func TestAttachDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}
	w.disks.m = map[string]*Resource{testDisk: {Project: testProject, RealName: testDisk, link: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}

	tests := []struct {
		desc    string
		ads     *AttachDisks
		wantErr bool
	}{
		{"blank case", &AttachDisks{}, false},
		{"normal case", &AttachDisks{{Instance: testInstance, AttachedDisk: compute.AttachedDisk{Mode: diskModeRW, Source: testDisk}}}, false},
		{"bad case", &AttachDisks{{Instance: "bad"}}, true},
	}
	for _, tt := range tests {
		err := tt.ads.run(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}
