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
	var url string
	listOpts := []ListCallOption{Filter("foo"), OrderBy("foo")}
	_, c, _ := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realCalled = true
		url = r.URL.String()
		w.WriteHeader(400)
		fmt.Fprintln(w, "Not Implemented")
	}))

	tests := []struct {
		desc string
		op   func()
		wURL string
	}{
		{"retry", func() {
			c.Retry(func(_ ...googleapi.CallOption) (*compute.Operation, error) { realCalled = true; return nil, nil })
		}, ""},
		{"attach disk", func() { c.AttachDisk("a", "b", "c", &compute.AttachedDisk{}) }, "/a/zones/b/instances/c/attachDisk?alt=json"},
		{"create disk", func() { c.CreateDisk("a", "b", &compute.Disk{}) }, "/a/zones/b/disks?alt=json"},
		{"create image", func() { c.CreateImage("a", &compute.Image{}) }, "/a/global/images?alt=json"},
		{"create instance", func() { c.CreateInstance("a", "b", &compute.Instance{}) }, "/a/zones/b/instances?alt=json"},
		{"create network", func() { c.CreateNetwork("a", &compute.Network{}) }, "/a/global/networks?alt=json"},
		{"instances stop", func() { c.StopInstance("a", "b", "c") }, "/a/zones/b/instances/c/stop?alt=json"},
		{"delete disk", func() { c.DeleteDisk("a", "b", "c") }, "/a/zones/b/disks/c?alt=json"},
		{"delete image", func() { c.DeleteImage("a", "b") }, "/a/global/images/b?alt=json"},
		{"delete instance", func() { c.DeleteInstance("a", "b", "c") }, "/a/zones/b/instances/c?alt=json"},
		{"delete network", func() { c.DeleteNetwork("a", "b") }, "/a/global/networks/b?alt=json"},
		{"deprecate image", func() { c.DeprecateImage("a", "b", &compute.DeprecationStatus{}) }, "/a/global/images/b/deprecate?alt=json"},
		{"get serial port", func() { c.GetSerialPortOutput("a", "b", "c", 1, 2) }, "/a/zones/b/instances/c/serialPort?alt=json&port=1&start=2"},
		{"get project", func() { c.GetProject("a") }, "/a?alt=json"},
		{"get machine type", func() { c.GetMachineType("a", "b", "c") }, "/a/zones/b/machineTypes/c?alt=json"},
		{"list machine types", func() { c.ListMachineTypes("a", "b", listOpts...) }, "/a/zones/b/machineTypes?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"get zone", func() { c.GetZone("a", "b") }, "/a/zones/b?alt=json"},
		{"list zones", func() { c.ListZones("a", listOpts...) }, "/a/zones?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"get instance", func() { c.GetInstance("a", "b", "c") }, "/a/zones/b/instances/c?alt=json"},
		{"list instances", func() { c.ListInstances("a", "b", listOpts...) }, "/a/zones/b/instances?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"get image from family", func() { c.GetImageFromFamily("a", "b") }, "/a/global/images/family/b?alt=json"},
		{"get image", func() { c.GetImage("a", "b") }, "/a/global/images/b?alt=json"},
		{"list images", func() { c.ListImages("a", listOpts...) }, "/a/global/images?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"get license", func() { c.GetLicense("a", "b") }, "/a/global/licenses/b?alt=json"},
		{"get network", func() { c.GetNetwork("a", "b") }, "/a/global/networks/b?alt=json"},
		{"list networks", func() { c.ListNetworks("a", listOpts...) }, "/a/global/networks?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"get disk", func() { c.GetDisk("a", "b", "c") }, "/a/zones/b/disks/c?alt=json"},
		{"list disks", func() { c.ListDisks("a", "b", listOpts...) }, "/a/zones/b/disks?alt=json&filter=foo&orderBy=foo&pageToken="},
		{"instance status", func() { c.InstanceStatus("a", "b", "c") }, "/a/zones/b/instances/c?alt=json"},
		{"instance stopped", func() { c.InstanceStopped("a", "b", "c") }, "/a/zones/b/instances/c?alt=json"},
		{"set instance metadata", func() { c.SetInstanceMetadata("a", "b", "c", nil) }, "/a/zones/b/instances/c/setMetadata?alt=json"},
		{"operation wait", func() { c.operationsWait("a", "b", "c") }, "/a/zones/b/operations/c?alt=json"},
	}

	runTests := func() {
		for _, tt := range tests {
			fakeCalled = false
			realCalled = false
			url = ""
			tt.op()
			if fakeCalled != wantFakeCalled || realCalled != wantRealCalled {
				t.Errorf("%s case: incorrect calls: wanted fakeCalled=%t realCalled=%t; got fakeCalled=%t realCalled=%t", tt.desc, wantFakeCalled, wantRealCalled, fakeCalled, realCalled)
			}
			if wantRealCalled {
				if tt.wURL != url {
					t.Errorf("%s case: want called url not equal to actual url, want: %q, got: %q", tt.desc, tt.wURL, url)
				}
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
	c.AttachDiskFn = func(_, _, _ string, _ *compute.AttachedDisk) error { fakeCalled = true; return nil }
	c.CreateDiskFn = func(_, _ string, _ *compute.Disk) error { fakeCalled = true; return nil }
	c.CreateImageFn = func(_ string, _ *compute.Image) error { fakeCalled = true; return nil }
	c.CreateInstanceFn = func(_, _ string, _ *compute.Instance) error { fakeCalled = true; return nil }
	c.CreateNetworkFn = func(_ string, _ *compute.Network) error { fakeCalled = true; return nil }
	c.StopInstanceFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	c.DeleteDiskFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	c.DeleteImageFn = func(_, _ string) error { fakeCalled = true; return nil }
	c.DeleteInstanceFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	c.DeleteNetworkFn = func(_, _ string) error { fakeCalled = true; return nil }
	c.DeprecateImageFn = func(_, _ string, _ *compute.DeprecationStatus) error { fakeCalled = true; return nil }
	c.GetSerialPortOutputFn = func(_, _, _ string, _, _ int64) (*compute.SerialPortOutput, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetProjectFn = func(_ string) (*compute.Project, error) { fakeCalled = true; return nil, nil }
	c.GetZoneFn = func(_, _ string) (*compute.Zone, error) { fakeCalled = true; return nil, nil }
	c.ListZonesFn = func(_ string, _ ...ListCallOption) ([]*compute.Zone, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetInstanceFn = func(_, _, _ string) (*compute.Instance, error) { fakeCalled = true; return nil, nil }
	c.ListInstancesFn = func(_, _ string, _ ...ListCallOption) ([]*compute.Instance, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetDiskFn = func(_, _, _ string) (*compute.Disk, error) { fakeCalled = true; return nil, nil }
	c.ListDisksFn = func(_, _ string, _ ...ListCallOption) ([]*compute.Disk, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetImageFromFamilyFn = func(_, _ string) (*compute.Image, error) { fakeCalled = true; return nil, nil }
	c.GetImageFn = func(_, _ string) (*compute.Image, error) { fakeCalled = true; return nil, nil }
	c.ListImagesFn = func(_ string, _ ...ListCallOption) ([]*compute.Image, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetLicenseFn = func(_, _ string) (*compute.License, error) { fakeCalled = true; return nil, nil }
	c.GetNetworkFn = func(_, _ string) (*compute.Network, error) { fakeCalled = true; return nil, nil }
	c.ListNetworksFn = func(_ string, _ ...ListCallOption) ([]*compute.Network, error) {
		fakeCalled = true
		return nil, nil
	}
	c.GetMachineTypeFn = func(_, _, _ string) (*compute.MachineType, error) { fakeCalled = true; return nil, nil }
	c.ListMachineTypesFn = func(_, _ string, _ ...ListCallOption) ([]*compute.MachineType, error) {
		fakeCalled = true
		return nil, nil
	}
	c.InstanceStatusFn = func(_, _, _ string) (string, error) { fakeCalled = true; return "", nil }
	c.InstanceStoppedFn = func(_, _, _ string) (bool, error) { fakeCalled = true; return false, nil }
	c.SetInstanceMetadataFn = func(_, _, _ string, _ *compute.Metadata) error { fakeCalled = true; return nil }
	c.operationsWaitFn = func(_, _, _ string) error { fakeCalled = true; return nil }
	wantFakeCalled = true
	wantRealCalled = false
	runTests()
}
