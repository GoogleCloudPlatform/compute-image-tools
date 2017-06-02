package compute

import (
	"context"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"net/http"
	"net/http/httptest"
)

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

type TestClient struct {
	client
	CreateDiskFn             func(project, zone string, d *compute.Disk) error
	CreateImageFn            func(project string, i *compute.Image) error
	CreateInstanceFn         func(project, zone string, i *compute.Instance) error
	DeleteDiskFn             func(project, zone, name string) error
	DeleteImageFn            func(project, name string) error
	DeleteInstanceFn         func(project, zone, name string) error
	GetSerialPortOutputFn    func(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	InstanceStatusFn         func(project, zone, name string) (string, error)
	InstanceStoppedFn        func(project, zone, name string) (bool, error)
	NewInstanceFn            func(name, project, zone, machineType string) (*Instance, error)
	WaitForInstanceStoppedFn func(project, zone, name string, interval time.Duration) error

	operationsWaitFn func(project, zone, name string) error
}

func (c *TestClient) CreateDisk(project, zone string, d *compute.Disk) error {
	if c.CreateDiskFn != nil {
		return c.CreateDiskFn(project, zone, d)
	}
	return c.client.CreateDisk(project, zone, d)
}

func (c *TestClient) CreateImage(project string, i *compute.Image) error {
	if c.CreateImageFn != nil {
		return c.CreateImageFn(project, i)
	}
	return c.client.CreateImage(project, i)
}

func (c *TestClient) CreateInstance(project, zone string, i *compute.Instance) error {
	if c.CreateImageFn != nil {
		return c.CreateInstanceFn(project, zone, i)
	}
	return c.client.CreateInstance(project, zone, i)
}

func (c *TestClient) DeleteDisk(project, zone, name string) error {
	if c.DeleteDiskFn != nil {
		return c.DeleteDiskFn(project, zone, name)
	}
	return c.client.DeleteDisk(project, zone, name)
}

func (c *TestClient) DeleteImage(project, name string) error {
	if c.DeleteImageFn != nil {
		return c.DeleteImageFn(project, name)
	}
	return c.client.DeleteImage(project, name)
}

func (c *TestClient) DeleteInstance(project, zone, name string) error {
	if c.DeleteInstanceFn != nil {
		return c.DeleteInstanceFn(project, zone, name)
	}
	return c.client.DeleteInstance(project, zone, name)
}

func (c *TestClient) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	if c.GetSerialPortOutputFn != nil {
		return c.GetSerialPortOutputFn(project, zone, name, port, start)
	}
	return c.client.GetSerialPortOutput(project, zone, name, port, start)
}

func (c *TestClient) InstanceStatus(project, zone, name string) (string, error) {
	if c.InstanceStatusFn != nil {
		return c.InstanceStatusFn(project, zone, name)
	}
	return c.client.InstanceStatus(project, zone, name)
}

func (c *TestClient) InstanceStopped(project, zone, name string) (bool, error) {
	if c.InstanceStoppedFn != nil {
		return c.InstanceStoppedFn(project, zone, name)
	}
	return c.client.InstanceStopped(project, zone, name)
}

func (c *TestClient) NewInstance(name, project, zone, machineType string) (*Instance, error) {
	if c.NewInstanceFn != nil {
		return c.NewInstanceFn(name, project, zone, machineType)
	}
	return c.client.NewInstance(name, project, zone, machineType)
}

func (c *TestClient) WaitForInstanceStopped(project, zone, name string, interval time.Duration) error {
	if c.WaitForInstanceStoppedFn != nil {
		return c.WaitForInstanceStoppedFn(project, zone, name, interval)
	}
	return c.client.WaitForInstanceStopped(project, zone, name, interval)
}

func (c *TestClient) operationsWait(project, zone, name string) error {
	if c.operationsWaitFn != nil {
		return c.operationsWaitFn(project, zone, name)
	}
	return c.client.operationsWait(project, zone, name)
}
