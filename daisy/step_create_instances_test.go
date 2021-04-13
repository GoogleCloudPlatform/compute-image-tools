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
	"errors"
	"testing"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"github.com/stretchr/testify/assert"
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func TestLogSerialOutput(t *testing.T) {
	ctx := context.Background()
	w := testWorkflow()
	w.instances.m = map[string]*Resource{
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

	tests := []struct {
		test, wantMessage1, wantMessage2 string
		instance                         *Instance
		instanceAlpha                    *InstanceAlpha
		instanceBeta                     *InstanceBeta
	}{
		{
			"Error but instance stopped",
			"Streaming instance \"i1\" serial port 0 output to https://storage.cloud.google.com/test-bucket/i1-serial-port0.log",
			"",
			&Instance{Instance: compute.Instance{Name: "i1"}},
			&InstanceAlpha{Instance: computeAlpha.Instance{Name: "i1"}},
			&InstanceBeta{Instance: computeBeta.Instance{Name: "i1"}},
		},
		{
			"Error but instance running",
			"Streaming instance \"i2\" serial port 0 output to https://storage.cloud.google.com/test-bucket/i2-serial-port0.log",
			"Instance \"i2\": error getting serial port: fail",
			&Instance{Instance: compute.Instance{Name: "i2"}},
			&InstanceAlpha{Instance: computeAlpha.Instance{Name: "i2"}},
			&InstanceBeta{Instance: computeBeta.Instance{Name: "i2"}},
		},
		{
			"Normal flow",
			"Streaming instance \"i3\" serial port 0 output to https://storage.cloud.google.com/test-bucket/i3-serial-port0.log",
			"",
			&Instance{Instance: compute.Instance{Name: "i3"}},
			&InstanceAlpha{Instance: computeAlpha.Instance{Name: "i3"}},
			&InstanceBeta{Instance: computeBeta.Instance{Name: "i3"}},
		},
		{
			"Error but instance deleted",
			"Streaming instance \"i4\" serial port 0 output to https://storage.cloud.google.com/test-bucket/i4-serial-port0.log",
			"",
			&Instance{Instance: compute.Instance{Name: "i4"}},
			&InstanceAlpha{Instance: computeAlpha.Instance{Name: "i4"}},
			&InstanceBeta{Instance: computeBeta.Instance{Name: "i4"}},
		},
	}

	assertTest := func(ctx context.Context, ii InstanceInterface, ib *InstanceBase, test, wantMessage1, wantMessage2 string) {
		mockLogger := &MockLogger{}
		w.Logger = mockLogger
		s := &Step{name: "foo", w: w}
		logSerialOutput(ctx, s, ii, ib, 0, 1*time.Microsecond)
		logEntries := mockLogger.getEntries()
		gotStep := logEntries[0].StepName
		if gotStep != "foo" {
			t.Errorf("%s: got: %q, want: %q", test, gotStep, "foo")
		}
		gotMessage := logEntries[0].Message
		if gotMessage != wantMessage1 {
			t.Errorf("%s: got: %q, want: %q", test, gotMessage, wantMessage1)
		}
		if wantMessage2 != "" {
			gotMessage := logEntries[1].Message
			if gotMessage != wantMessage2 {
				t.Errorf("%s: got: %q, want: %q", test, gotMessage, wantMessage2)
			}
		}
	}
	for _, tt := range tests {
		assertTest(ctx, tt.instance, &tt.instance.InstanceBase, tt.test, tt.wantMessage1, tt.wantMessage2)
	}
}

func TestLogSerialOutputStopsAfterTenRetries(t *testing.T) {

	testSerialOutput := func(ii InstanceInterface, ib *InstanceBase) {
		w := testWorkflow()

		callNum := 0
		responses := []string{"", "", "hello", "", " go", "", "", "", "", "", "", "", "", "", "", "", "lang"}
		nexts := []int64{0, 0, 0, 5, 5, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}

		w.ComputeClient.(*daisyCompute.TestClient).GetSerialPortOutputFn = func(_, _, n string, _, next int64) (*compute.SerialPortOutput, error) {
			response := responses[callNum]
			assert.Equal(t, nexts[callNum], next)
			callNum++
			switch response {
			case "":
				return nil, errors.New("fail")
			default:
				return &compute.SerialPortOutput{Contents: response, Next: next + int64(len(response))}, nil
			}
		}
		logSerialOutput(context.Background(), &Step{name: "foo", w: w}, ii, ib, 0, 1*time.Microsecond)
		logs := w.Logger.ReadSerialPortLogs()
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "hello go", logs[0])
	}
	i := Instance{Instance: compute.Instance{Name: "i1"}}
	testSerialOutput(&i, &i.InstanceBase)

	iAlpha := InstanceAlpha{Instance: computeAlpha.Instance{Name: "i1Alpha"}}
	testSerialOutput(&iAlpha, &iAlpha.InstanceBase)

	iBeta := InstanceBeta{Instance: computeBeta.Instance{Name: "i1Beta"}}
	testSerialOutput(&iBeta, &iBeta.InstanceBase)
}

func TestCreateInstancesRun(t *testing.T) {
	ctx := context.Background()
	var createErr DError
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceFn = func(p, z string, i *compute.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceAlphaFn = func(p, z string, i *computeAlpha.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceBetaFn = func(p, z string, i *computeBeta.Instance) error {
		i.SelfLink = "insertedLink"
		return createErr
	}
	s := &Step{w: w}
	w.Sources = map[string]string{"file": "gs://some/file"}
	w.disks.m = map[string]*Resource{"d": {link: "dLink"}}
	w.networks.m = map[string]*Resource{"n": {link: "nLink"}}
	w.subnetworks.m = map[string]*Resource{"s": {link: "sLink"}}

	// Good case: check disk and network links get resolved.
	i0 := &Instance{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i0"}}, Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d"}}, NetworkInterfaces: []*compute.NetworkInterface{{Network: "n"}}}}
	i1 := &Instance{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i1", Project: "foo"}}, Instance: compute.Instance{Name: "realI1", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "other"}}, Zone: "bar"}}
	i2 := &Instance{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i2"}}, Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d"}}, NetworkInterfaces: []*compute.NetworkInterface{{Subnetwork: "s"}}}}
	i0Beta := &InstanceBeta{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i0"}}, Instance: computeBeta.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*computeBeta.AttachedDisk{{Source: "d"}}, NetworkInterfaces: []*computeBeta.NetworkInterface{{Network: "n"}}}}
	i1Beta := &InstanceBeta{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i1", Project: "foo"}}, Instance: computeBeta.Instance{Name: "realI1", MachineType: "foo-type", Disks: []*computeBeta.AttachedDisk{{Source: "other"}}, Zone: "bar"}}
	i2Beta := &InstanceBeta{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i2"}}, Instance: computeBeta.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*computeBeta.AttachedDisk{{Source: "d"}}, NetworkInterfaces: []*computeBeta.NetworkInterface{{Subnetwork: "s"}}}}

	ci := &CreateInstances{Instances: []*Instance{i0, i1, i2}, InstancesBeta: []*InstanceBeta{i0Beta, i1Beta, i2Beta}}

	if err := ci.run(ctx, s); err != nil {
		t.Errorf("unexpected error running CreateInstances.run(): %v", err)
	}
	if i0.Disks[0].Source != w.disks.m["d"].link {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", w.disks.m["d0"].link, i0.Disks[0].Source)
	}
	if i0.NetworkInterfaces[0].Network != w.networks.m["n"].link {
		t.Errorf("instance network link did not resolve properly: want: %q, got: %q", w.networks.m["n"].link, i0.NetworkInterfaces[0].Network)
	}
	if i1.Disks[0].Source != "other" {
		t.Errorf("instance disk link did not resolve properly: want: %q, got: %q", "other", i1.Disks[0].Source)
	}
	if i2.NetworkInterfaces[0].Subnetwork != w.subnetworks.m["s"].link {
		t.Errorf("instance network link did not resolve properly: want: %q, got: %q", w.subnetworks.m["s"].link, i2.NetworkInterfaces[0].Network)
	}

	// Bad case: compute client Instance error.
	w.instances.m = map[string]*Resource{}
	createErr = Errf("client error")
	ci = &CreateInstances{
		Instances: []*Instance{
			{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i0"}}, Instance: compute.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*compute.AttachedDisk{{Source: "d0"}}}},
		},
	}
	if err := ci.run(ctx, s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
	ci = &CreateInstances{
		InstancesBeta: []*InstanceBeta{
			{InstanceBase: InstanceBase{Resource: Resource{daisyName: "i0"}}, Instance: computeBeta.Instance{Name: "realI0", MachineType: "foo-type", Disks: []*computeBeta.AttachedDisk{{Source: "d0"}}}},
		},
	}
	if err := ci.run(ctx, s); err != createErr {
		t.Errorf("CreateInstances.run() should have return compute client error: %v != %v", err, createErr)
	}
}
