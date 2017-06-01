package compute

import (
	"context"
	"time"

	"google.golang.org/api/compute/v1"
	"net/http"
	"net/http/httptest"
	"google.golang.org/api/option"
	"fmt"
)

var (
	testProject  = "test-project"
	testZone     = "test-zone"
	testDisk     = "test-disk"
	testImage    = "test-image"
	testInstance = "test-instance"
)

func NewTestClient(handleFunc http.HandlerFunc) (*httptest.Server, *TestClient, error) {
	tc := &TestClient{}
	ts := httptest.NewServer(handleFunc)
	c, err := NewClient(context.Background(), option.WithEndpoint(ts.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		return nil, nil, err
	}

	rc := c.(*realClient)
	tc.realClient = *rc
	return ts, tc, nil
}

type TestClient struct {
	realClient
	CreateDiskFn func (project, zone string, d *compute.Disk) error
	CreateImageFn func (project string, i *compute.Image) error
	DeleteDiskFn func (project, zone, name string) error
	DeleteImageFn func (project, name string) error
	DeleteInstanceFn func (project, zone, name string) error
	GetSerialPortOutputFn func (project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	InstanceStatusFn func (project, zone, name string) (string, error)
	InstanceStoppedFn func (project, zone, name string) (bool, error)
	WaitForInstanceStoppedFn func (project, zone, name string, interval time.Duration) error

	operationsWaitFn func(project, zone, name string) error
}

func (c *TestClient) CreateDisk(project, zone string, d *compute.Disk) error {
	if c.CreateDiskFn != nil {
		return c.CreateDiskFn(project, zone, d)
	}
	return c.realClient.CreateDisk(project, zone, d)
}

func (c *TestClient) CreateImage(project string, i *compute.Image) error {
	if c.CreateImageFn != nil {
		return c.CreateImageFn(project, i)
	}
	return c.realClient.CreateImage(project, i)
}

func (c *TestClient) DeleteDisk(project, zone, name string) error {
	if c.DeleteDiskFn != nil {
		return c.DeleteDiskFn(project, zone, name)
	}
	return c.realClient.DeleteDisk(project, zone, name)
}

func (c *TestClient) DeleteImage(project, name string) error {
	if c.DeleteImageFn != nil {
		return c.DeleteImageFn(project, name)
	}
	return c.realClient.DeleteImage(project, name)
}

func (c *TestClient) DeleteInstance(project, zone, name string) error {
	if c.DeleteInstanceFn != nil {
		return c.DeleteInstanceFn(project, zone, name)
	}
	return c.realClient.DeleteInstance(project, zone, name)
}

func (c *TestClient) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	if c.GetSerialPortOutputFn != nil {
		return c.GetSerialPortOutputFn(project, zone, name, port, start)
	}
	return c.realClient.GetSerialPortOutput(project, zone, name, port, start)
}

func (c *TestClient) InstanceStatus(project, zone, name string) (string, error) {
	if c.InstanceStatusFn != nil {
		return c.InstanceStatusFn(project, zone, name)
	}
	return c.realClient.InstanceStatus(project, zone, name)
}

func (c *TestClient) InstanceStopped(project, zone, name string) (bool, error) {
	if c.InstanceStoppedFn != nil {
		return c.InstanceStoppedFn(project, zone, name)
	}
	return c.realClient.InstanceStopped(project, zone, name)
}

func (c *TestClient) WaitForInstanceStopped(project, zone, name string, interval time.Duration) error {
	if c.WaitForInstanceStoppedFn != nil {
		return c.WaitForInstanceStoppedFn(project, zone, name, interval)
	}
	return c.realClient.WaitForInstanceStopped(project, zone, name, interval)
}

func (c *TestClient) operationsWait(project, zone, name string) error {
	fmt.Println("HERE1!")
	if c.operationsWaitFn != nil {
		fmt.Println("HERE2!")
		return c.operationsWaitFn(project, zone, name)
	}
	fmt.Println("HERE3!")
	return c.realClient.operationsWait(project, zone, name)
}