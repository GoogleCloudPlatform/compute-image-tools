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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

func TestLogSerialOutput(t *testing.T) {
	w := testWorkflow()
	w.instances.m = map[string]*Resource{
		"i1": {RealName: w.genName("i1"), link: "link"},
		"i2": {RealName: w.genName("i2"), link: "link"},
		"i3": {RealName: w.genName("i3"), link: "link"},
	}

	mockLogger := &MockLogger{}
	w.Logger = mockLogger
	s := &Step{name: "i1", w: w}
	mockWatcher := newMockSerialOutputWatcher(t, []string{"log ", "content"})
	mockWriter := mockStorageClient{}
	logSerialOutput(s, "instance-name", &mockWatcher, func() io.WriteCloser {
		return &mockWriter
	})
	assert.Equal(t, "log content", mockWriter.content)
}

func TestCreateInstancesRun(t *testing.T) {
	ctx := context.Background()
	var createErr DError
	w := testWorkflow()
	w.ComputeClient.(*daisyCompute.TestClient).CreateInstanceFn = func(p, z string, i *compute.Instance) error {
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

func newMockSerialOutputWatcher(t *testing.T, responses []string) SerialOutputWatcher {
	return &mockSerialOutputWatcher{t: t, responses: responses}
}

type mockSerialOutputWatcher struct {
	t            *testing.T
	instanceName string
	c            chan<- string
	responses    []string
}

func (m *mockSerialOutputWatcher) Watch(instanceName string, port int64, c chan<- string, interval time.Duration) {
	m.instanceName = instanceName
	m.c = c
}

func (m *mockSerialOutputWatcher) start(instanceName string) {
	assert.Equal(m.t, m.instanceName, instanceName)
	for _, response := range m.responses {
		m.c <- response
	}
	close(m.c)
}

type mockStorageClient struct {
	numWrite int
	numClose int
	content  string
}

func (m *mockStorageClient) Write(p []byte) (n int, err error) {
	m.numWrite++
	m.content += string(p)
	return len(p), nil
}

func (m mockStorageClient) Close() error {
	m.numClose++
	return nil
}
