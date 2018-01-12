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
	"bytes"
	"context"
	"errors"
	"log"
	"testing"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/compute/v1"
)

func TestLogSerialOutput(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	instances[w].m = map[string]*Resource{
		"i1": {RealName: w.genName("i1"), link: "link"},
		"i2": {RealName: w.genName("i2"), link: "link"},
		"i3": {RealName: w.genName("i3"), link: "link"},
	}

	w.ComputeClient.(*daisyCompute.TestClient).GetSerialPortOutputFn = func(_, _, n string, _, s int64) (*compute.SerialPortOutput, error) {
		if n == "i3" && s == 0 {
			return &compute.SerialPortOutput{Contents: "", Next: 1}, nil
		}
		return nil, errors.New("fail")
	}
	w.ComputeClient.(*daisyCompute.TestClient).InstanceStoppedFn = func(_, _, n string) (bool, error) {
		if n == "i2" {
			return false, nil
		}
		return true, nil
	}

	w.bucket = "test-bucket"

	var buf bytes.Buffer
	w.Logger = log.New(&buf, "", 0)

	tests := []struct {
		test, want, name string
	}{
		{
			"Error but instance stopped",
			"CreateInstances: streaming instance \"i1\" serial port 0 output to gs://test-bucket/i1-serial-port0.log\n",
			"i1",
		},
		{
			"Error but instance running",
			"CreateInstances: streaming instance \"i2\" serial port 0 output to gs://test-bucket/i2-serial-port0.log\nCreateInstances: instance \"i2\": error getting serial port: fail\n",
			"i2",
		},
		{
			"Normal flow",
			"CreateInstances: streaming instance \"i3\" serial port 0 output to gs://test-bucket/i3-serial-port0.log\n",
			"i3",
		},
		{
			"Error but instance deleted",
			"CreateInstances: streaming instance \"i4\" serial port 0 output to gs://test-bucket/i4-serial-port0.log\n",
			"i4",
		},
	}

	for _, tt := range tests {
		buf.Reset()
		logSerialOutput(ctx, w, tt.name, 0, 1*time.Microsecond)
		if buf.String() != tt.want {
			t.Errorf("%s: got: %q, want: %q", tt.test, buf.String(), tt.want)
		}
	}
}

func TestCreateInstancesRun(t *testing.T) {
	ctx := context.Background()
	var createErr dErr
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceFn = func(p, z string, i *compute.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	s := &Step{w: w}
	w.Sources = map[string]string{"file": "gs://some/file"}
	disks[w].m = map[string]*Resource{
		"d0": {RealName: w.genName("d0"), link: "diskLink0"},
	}

	// Good case: check disk link gets resolved. Check instance reference map updates.
	i0 := &Instance{Resource: Resource{daisyName: "i0"}, Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}}
	i1 := &Instance{Resource: Resource{daisyName: "i1", Project: "foo"}, Instance: compute.Instance{Name: "realI1", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "other"}}, Zone: "bar"}}
	ci := &CreateInstances{i0, i1}
	if err := ci.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateInstances.run(): %v", err)
	}
	if i0.Disks[0].Source != disks[w].m["d0"].link {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", disks[w].m["d0"].link, i0.Disks[0].Source)
	}
	if i1.Disks[0].Source != "other" {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", "other", i1.Disks[0].Source)
	}

	// Bad case: compute client Instance error. Check instance ref map doesn't update.
	instances[w].m = map[string]*Resource{}
	createErr = errf("client error")
	ci = &CreateInstances{
		{Resource: Resource{daisyName: "i0"}, Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}},
	}
	if err := ci.run(ctx, s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
}
