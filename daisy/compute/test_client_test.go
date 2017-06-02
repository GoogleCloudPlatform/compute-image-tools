package compute

import (
	"fmt"
	"google.golang.org/api/compute/v1"
	"net/http"
	"testing"
	"time"
)

func TestTestClient(t *testing.T) {
	var fakeCalled, realCalled bool
	var wantFakeCalled, wantRealCalled bool
	_, c, _ := NewTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realCalled = true
		w.WriteHeader(500)
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
		{"instance status", func() { c.InstanceStatus("a", "b", "c") }},
		{"instance stopped", func() { c.InstanceStopped("a", "b", "c") }},
		{"wait instance", func() { c.WaitForInstanceStopped("a", "b", "c", time.Duration(1)) }},
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
	c.CreateDiskFn = func(p, z string, d *compute.Disk) error { fakeCalled = true; return nil }
	c.CreateImageFn = func(p string, i *compute.Image) error { fakeCalled = true; return nil }
	c.CreateInstanceFn = func(p, z string, i *compute.Instance) error { fakeCalled = true; return nil }
	c.DeleteDiskFn = func(p, z, n string) error { fakeCalled = true; return nil }
	c.DeleteImageFn = func(p, n string) error { fakeCalled = true; return nil }
	c.DeleteInstanceFn = func(p, z, n string) error { fakeCalled = true; return nil }
	c.GetSerialPortOutputFn = func(p, z, n string, port, start int64) (*compute.SerialPortOutput, error) {
		fakeCalled = true
		return nil, nil
	}
	c.InstanceStatusFn = func(p, z, n string) (string, error) { fakeCalled = true; return "", nil }
	c.InstanceStoppedFn = func(p, z, n string) (bool, error) { fakeCalled = true; return false, nil }
	c.WaitForInstanceStoppedFn = func(p, z, n string, i time.Duration) error { fakeCalled = true; return nil }
	c.operationsWaitFn = func(p, z, n string) error { fakeCalled = true; return nil }
	wantFakeCalled = true
	wantRealCalled = false
	runTests()
}
