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
)

func TestDetachDisksPopulate(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}

	tests := []struct {
		desc    string
		dds     *DetachDisks
		wantErr bool
	}{
		{"default", &DetachDisks{{DeviceName: "someDisk"}}, false},
		{"no name", &DetachDisks{{DeviceName: ""}}, false},
		{"match regex", &DetachDisks{{DeviceName: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}, false},
	}
	for _, tt := range tests {

		err := tt.dds.populate(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}

func TestDetachDisksValidate(t *testing.T) {
	ctx := context.Background()
	testDeviceURL := fmt.Sprintf("projects/%s/zones/%s/devices/%s", testProject, testZone, testDisk)
	testInstanceURL := fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)

	tests := []struct {
		desc             string
		attachDeviceName string
		dds              *DetachDisks
		wantErr          bool
	}{
		{"empty source case", "", &DetachDisks{{Instance: testInstance, DeviceName: ""}}, true},
		{"bad source case", "", &DetachDisks{{Instance: testInstance, DeviceName: "bad"}}, true},
		{"bad instance case", "", &DetachDisks{{Instance: "bad", DeviceName: testDisk}}, true},
		{"bad project and zone case", "", &DetachDisks{{Instance: testInstance, DeviceName: "projects/bad/zones/bad/devices/bad"}}, true},
		{"wrong url (disk url) case", "", &DetachDisks{{Instance: testInstance, DeviceName: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)}}, true},
		{"instance url + device url case (instance pre-exists, device pre-attached)", "", &DetachDisks{{Instance: testInstanceURL, DeviceName: testDeviceURL}}, false},
		{"instance url + device name case (instance pre-exists, device attached in workflow)", "", &DetachDisks{{Instance: testInstanceURL, DeviceName: testDisk}}, false},
		{"instance name + device name case", "", &DetachDisks{{Instance: testInstance, DeviceName: testDisk}}, false},
		{"device name != disk name case", "device-name", &DetachDisks{{Instance: testInstance, DeviceName: "device-name"}}, false},
	}
	for _, tt := range tests {
		w := testWorkflow()
		s, _ := w.NewStep("DetacherStep")
		w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: testInstanceURL}}
		w.disks.m = map[string]*Resource{
			testDisk: {Project: testProject, RealName: testDisk, link: fmt.Sprintf("projects/%s/zones/%s/disks/%s", testProject, testZone, testDisk)},
			"bad":    {Project: "bad", RealName: testDisk, link: "link"},
		}
		instanceRegisterName := testInstance
		if (*tt.dds)[0].Instance == testInstanceURL {
			instanceRegisterName = testInstanceURL
		}

		// Register the instance/disk to respect dependency
		as, _ := w.NewStep("AttacherStep")
		attachDeviceName := tt.attachDeviceName
		if attachDeviceName == "" {
			attachDeviceName = testDisk
		}
		w.instances.regURL(instanceRegisterName, false)
		w.disks.regAttach(attachDeviceName, testDisk, instanceRegisterName, "", as)
		w.AddDependency(s, as)

		err := tt.dds.validate(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}

func TestDetachDisksRun(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	s := &Step{w: w}
	w.instances.m = map[string]*Resource{testInstance: {Project: testProject, RealName: testInstance, link: fmt.Sprintf("projects/%s/zones/%s/instances/%s", testProject, testZone, testInstance)}}

	tests := []struct {
		desc    string
		dds     *DetachDisks
		wantErr bool
	}{
		{"blank case", &DetachDisks{}, false},
		{"normal case", &DetachDisks{{Instance: testInstance, DeviceName: testDisk}}, false},
		{"bad case", &DetachDisks{{Instance: "bad"}}, true},
	}
	for _, tt := range tests {
		err := tt.dds.run(ctx, s)
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		}
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error, got none", tt.desc)
		}
	}
}
