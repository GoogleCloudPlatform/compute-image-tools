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
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// NewTestClient returns a TestClient with a replacement http handler function.
// Methods on the new TestClient are overrideable as well.
func NewTestClient(handleFunc http.HandlerFunc) (*httptest.Server, *TestClient, error) {
	ts := httptest.NewServer(handleFunc)
	opts := []option.ClientOption{
		option.WithEndpoint(ts.URL),
		option.WithHTTPClient(http.DefaultClient),
	}
	c, err := NewClient(context.Background(), opts...)
	if err != nil {
		return nil, nil, err
	}

	tc := &TestClient{}
	tc.client = *c.(*client)
	tc.client.i = tc
	return ts, tc, nil
}

// TestClient is a Client with overrideable methods.
type TestClient struct {
	client
	CreateDiskFn             func(project, zone string, d *compute.Disk) error
	CreateImageFn            func(project string, i *compute.Image) error
	CreateInstanceFn         func(project, zone string, i *compute.Instance) error
	DeleteDiskFn             func(project, zone, name string) error
	DeleteImageFn            func(project, name string) error
	DeleteInstanceFn         func(project, zone, name string) error
	GetMachineTypeFn         func(project, zone, machineType string) (*compute.MachineType, error)
	GetProjectFn             func(project string) (*compute.Project, error)
	GetSerialPortOutputFn    func(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	GetZoneFn                func(project, zone string) (*compute.Zone, error)
	InstanceStatusFn         func(project, zone, name string) (string, error)
	InstanceStoppedFn        func(project, zone, name string) (bool, error)
	WaitForInstanceStoppedFn func(project, zone, name string, interval time.Duration) error

	operationsWaitFn func(project, zone, name string) error
}

// CreateDisk uses the override method CreateDiskFn or the real implementation.
func (c *TestClient) CreateDisk(project, zone string, d *compute.Disk) error {
	if c.CreateDiskFn != nil {
		return c.CreateDiskFn(project, zone, d)
	}
	return c.client.CreateDisk(project, zone, d)
}

// CreateImage uses the override method CreateImageFn or the real implementation.
func (c *TestClient) CreateImage(project string, i *compute.Image) error {
	if c.CreateImageFn != nil {
		return c.CreateImageFn(project, i)
	}
	return c.client.CreateImage(project, i)
}

// CreateInstance uses the override method CreateInstanceFn or the real implementation.
func (c *TestClient) CreateInstance(project, zone string, i *compute.Instance) error {
	if c.CreateInstanceFn != nil {
		return c.CreateInstanceFn(project, zone, i)
	}
	return c.client.CreateInstance(project, zone, i)
}

// DeleteDisk uses the override method DeleteDiskFn or the real implementation.
func (c *TestClient) DeleteDisk(project, zone, name string) error {
	if c.DeleteDiskFn != nil {
		return c.DeleteDiskFn(project, zone, name)
	}
	return c.client.DeleteDisk(project, zone, name)
}

// DeleteImage uses the override method DeleteImageFn or the real implementation.
func (c *TestClient) DeleteImage(project, name string) error {
	if c.DeleteImageFn != nil {
		return c.DeleteImageFn(project, name)
	}
	return c.client.DeleteImage(project, name)
}

// DeleteInstance uses the override method DeleteInstanceFn or the real implementation.
func (c *TestClient) DeleteInstance(project, zone, name string) error {
	if c.DeleteInstanceFn != nil {
		return c.DeleteInstanceFn(project, zone, name)
	}
	return c.client.DeleteInstance(project, zone, name)
}

// GetProject uses the override method GetProjectFn or the real implementation.
func (c *TestClient) GetProject(project string) (*compute.Project, error) {
	if c.GetProjectFn != nil {
		return c.GetProjectFn(project)
	}
	return c.client.GetProject(project)
}

// GetMachineType uses the override method GetMachineTypeFn or the real implementation.
func (c *TestClient) GetMachineType(project, zone, machineType string) (*compute.MachineType, error) {
	if c.GetZoneFn != nil {
		return c.GetMachineTypeFn(project, zone, machineType)
	}
	return c.client.GetMachineType(project, zone, machineType)
}

// GetZone uses the override method GetZoneFn or the real implementation.
func (c *TestClient) GetZone(project, zone string) (*compute.Zone, error) {
	if c.GetZoneFn != nil {
		return c.GetZoneFn(project, zone)
	}
	return c.client.GetZone(project, zone)
}

// GetSerialPortOutput uses the override method GetSerialPortOutputFn or the real implementation.
func (c *TestClient) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	if c.GetSerialPortOutputFn != nil {
		return c.GetSerialPortOutputFn(project, zone, name, port, start)
	}
	return c.client.GetSerialPortOutput(project, zone, name, port, start)
}

// InstanceStatus uses the override method InstanceStatusFn or the real implementation.
func (c *TestClient) InstanceStatus(project, zone, name string) (string, error) {
	if c.InstanceStatusFn != nil {
		return c.InstanceStatusFn(project, zone, name)
	}
	return c.client.InstanceStatus(project, zone, name)
}

// InstanceStopped uses the override method InstanceStoppedFn or the real implementation.
func (c *TestClient) InstanceStopped(project, zone, name string) (bool, error) {
	if c.InstanceStoppedFn != nil {
		return c.InstanceStoppedFn(project, zone, name)
	}
	return c.client.InstanceStopped(project, zone, name)
}

// WaitForInstanceStopped uses the override method WaitForInstanceStoppedFn or the real implementation.
func (c *TestClient) WaitForInstanceStopped(project, zone, name string, interval time.Duration) error {
	if c.WaitForInstanceStoppedFn != nil {
		return c.WaitForInstanceStoppedFn(project, zone, name, interval)
	}
	return c.client.WaitForInstanceStopped(project, zone, name, interval)
}

// operationsWait uses the override method operationsWaitFn or the real implementation.
func (c *TestClient) operationsWait(project, zone, name string) error {
	if c.operationsWaitFn != nil {
		return c.operationsWaitFn(project, zone, name)
	}
	return c.client.operationsWait(project, zone, name)
}
