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

package compute

import (
	"fmt"
	"net/http"
	"testing"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func TestTestClient(t *testing.T) {
	var fakeCalled, realCalled bool
	var wantFakeCalled, wantRealCalled bool
	_, c, _ := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realCalled = true
		w.WriteHeader(400)
		fmt.Fprintln(w, "Not Implemented")
	}))

	tests := []struct {
		desc string
		op   func()
	}{
		{"create disk", func() { c.CreateDisk("a", "b", &compute.Disk{}) }},
		{"create image", func() { c.CreateImage("a", &compute.Image{}) }},
		{"create instance", func() { c.CreateInstance("a", "b", &compute.Instance{}) }},
		{"delete disk", func() { c.DeleteDisk("a", "b", "c") }},
		{"delete image", func() { c.DeleteImage("a", "b") }},
		{"delete instance", func() { c.DeleteInstance("a", "b", "c") }},
		{"get serial port", func() { c.GetSerialPortOutput("a", "b", "c", 1, 2) }},
		{"get project", func() { c.GetProject("a") }},
		{"get machine type", func() { c.GetMachineType("a", "b", "c") }},
		{"get zone", func() { c.GetZone("a", "b") }},
		{"instance status", func() { c.InstanceStatus("a", "b", "c") }},
		{"instance stopped", func() { c.InstanceStopped("a", "b", "c") }},
		{"operation wait", func() { c.operationsWait("a", "b", "c") }},
	}

	runTests := func() {
		for _, tt := range tests {
			fakeCalled = false
			realCalled = false
			tt.op()
			if fakeCalled != wantFakeCalled || realCalled != wantRealCalled {
				t.Errorf("%s case: incorrect calls: wanted fakeCalled=%t realCalled=%t; got fakeCalled=%t realCalled=%t", tt.desc, wantFakeCalled, wantRealCalled, fakeCalled, realCalled)
			}
		}
	}

	// Test real methods can be called.
	wantFakeCalled = false
	wantRealCalled = true
	runTests()

	// Test fake methods can be called.
	c.RetryFn = func(_ func(_ ...googleapi.CallOption) (*compute.Operation, error), _ ...googleapi.CallOption) (op *compute.Operation, err error) {
		fakeCalled = true
		return nil, nil
	}
	c.CreateDiskFn = func(_, _ string, _ *compute.Disk) error { fakeCalled = true; return nil }
	c.CreateImageFn = func(_ string, _ *compute.Image) error { fakeCalled = true; return nil }
	c.CreateInstanceFn = func(_, _ string, _ *compute.Instance) error { fakeCalled = true; return nil }
	c.DeleteDiskFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	c.DeleteImageFn = func(_, _ string) error { fakeCalled = true; return nil }
	c.DeleteInstanceFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	c.GetSerialPortOutputFn = func(_, _, _ string, _, _ int64) (*compute.SerialPortOutput, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetProjectFn = func(_ string) (*compute.Project, error) { fakeCalled = true; return nil, nil }
	c.GetZoneFn = func(_, _ string) (*compute.Zone, error) { fakeCalled = true; return nil, nil }
	c.GetInstanceFn = func(_, _, _ string) (*compute.Instance, error) { fakeCalled = true; return nil, nil }
	c.GetDiskFn = func(_, _, _ string) (*compute.Disk, error) { fakeCalled = true; return nil, nil }
	c.GetImageFn = func(_, _ string) (*compute.Image, error) { fakeCalled = true; return nil, nil }
	c.GetMachineTypeFn = func(_, _, _ string) (*compute.MachineType, error) { fakeCalled = true; return nil, nil }
	c.InstanceStatusFn = func(_, _, _ string) (string, error) { fakeCalled = true; return "", nil }
	c.InstanceStoppedFn = func(_, _, _ string) (bool, error) { fakeCalled = true; return false, nil }
	c.operationsWaitFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	wantFakeCalled = true
	wantRealCalled = false
	runTests()
}
