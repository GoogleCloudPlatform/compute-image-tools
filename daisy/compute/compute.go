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

// Package compute provides access to the Google Compute API.
package compute

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

// Client is a client for interacting with Google Cloud Compute.
type Client interface {
	CreateDisk(project, zone string, d *compute.Disk) error
	CreateImage(project string, i *compute.Image) error
	CreateInstance(project, zone string, i *compute.Instance) error
	DeleteDisk(project, zone, name string) error
	DeleteImage(project, name string) error
	DeleteInstance(project, zone, name string) error
	GetMachineType(project, zone, machineType string) (*compute.MachineType, error)
	ListMachineTypes(project, zone string) ([]*compute.MachineType, error)
	GetProject(project string) (*compute.Project, error)
	GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	GetZone(project, zone string) (*compute.Zone, error)
	ListZones(project string) ([]*compute.Zone, error)
	GetInstance(project, zone, name string) (*compute.Instance, error)
	ListInstances(project, zone string) ([]*compute.Instance, error)
	GetDisk(project, zone, name string) (*compute.Disk, error)
	ListDisks(project, zone string) ([]*compute.Disk, error)
	GetImage(project, name string) (*compute.Image, error)
	GetImageFromFamily(project, family string) (*compute.Image, error)
	ListImages(project string) ([]*compute.Image, error)
	GetLicense(project, name string) (*compute.License, error)
	GetNetwork(project, name string) (*compute.Network, error)
	ListNetworks(project string) ([]*compute.Network, error)
	InstanceStatus(project, zone, name string) (string, error)
	InstanceStopped(project, zone, name string) (bool, error)
	Retry(f func(opts ...googleapi.CallOption) (*compute.Operation, error), opts ...googleapi.CallOption) (op *compute.Operation, err error)
}

type clientImpl interface {
	Client
	operationsWait(project, zone, name string) error
}

type client struct {
	i   clientImpl
	hc  *http.Client
	raw *compute.Service
}

// shouldRetryWithWait returns true if the HTTP response / error indicates
// that the request should be attempted again.
func shouldRetryWithWait(tripper http.RoundTripper, err error, multiplier int) bool {
	if err == nil {
		return false
	}
	tkValid := true
	trans, ok := tripper.(*oauth2.Transport)
	if ok {
		if tk, err := trans.Source.Token(); err == nil {
			tkValid = tk.Valid()
		}
	}

	apiErr, ok := err.(*googleapi.Error)
	var retry bool
	switch {
	case !ok && tkValid:
		// Not a googleapi.Error and the token is still valid.
		return false
	case apiErr.Code >= 500 && apiErr.Code <= 599:
		retry = true
	case apiErr.Code >= 429:
		// Too many API requests.
		retry = true
	case !tkValid:
		// This was probably a failure to get new token from metadata server.
		retry = true
	}
	if !retry {
		return false
	}

	sleep := (time.Duration(rand.Intn(1000))*time.Millisecond + 1*time.Second) * time.Duration(multiplier)
	time.Sleep(sleep)
	return true
}

// NewClient creates a new Google Cloud Compute client.
func NewClient(ctx context.Context, opts ...option.ClientOption) (Client, error) {
	o := []option.ClientOption{option.WithScopes(compute.ComputeScope)}
	opts = append(o, opts...)
	hc, ep, err := transport.NewHTTPClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("dialing: %v", err)
	}
	rawService, err := compute.New(hc)
	if err != nil {
		return nil, fmt.Errorf("compute client: %v", err)
	}
	if ep != "" {
		rawService.BasePath = ep
	}
	c := &client{hc: hc, raw: rawService}
	c.i = c

	return c, nil
}

func (c *client) operationsWait(project, zone, name string) error {
	for {
		var err error
		var op *compute.Operation
		if zone != "" {
			op, err = c.Retry(c.raw.ZoneOperations.Get(project, zone, name).Do)
			if err != nil {
				return fmt.Errorf("failed to get operation %s: %v", name, err)
			}
		} else {
			op, err = c.Retry(c.raw.GlobalOperations.Get(project, name).Do)
			if err != nil {
				return fmt.Errorf("failed to get operation %s: %v", name, err)
			}
		}
		switch op.Status {
		case "PENDING", "RUNNING":
			time.Sleep(1 * time.Second)
			continue
		case "DONE":
			if op.Error != nil {
				var operrs string
				for _, operr := range op.Error.Errors {
					operrs = operrs + fmt.Sprintf("\n  Code: %s, Message: %s", operr.Code, operr.Message)
				}
				return fmt.Errorf("operation failed %+v: %s", op, operrs)
			}
		default:
			return fmt.Errorf("unknown operation status %q: %+v", op.Status, op)
		}
		return nil
	}
}

// Retry invokes the given function, retrying it multiple times if the HTTP
// status response indicates the request should be attempted again or the
// oauth Token is no longer valid.
func (c *client) Retry(f func(opts ...googleapi.CallOption) (*compute.Operation, error), opts ...googleapi.CallOption) (op *compute.Operation, err error) {
	for i := 1; i < 4; i++ {
		op, err = f(opts...)
		if err == nil {
			return op, nil
		}
		if !shouldRetryWithWait(c.hc.Transport, err, i) {
			return nil, err
		}
	}
	return
}

// CreateDisk creates a GCE persistent disk.
func (c *client) CreateDisk(project, zone string, d *compute.Disk) error {
	op, err := c.Retry(c.raw.Disks.Insert(project, zone, d).Do)
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, zone, op.Name); err != nil {
		return err
	}

	var createdDisk *compute.Disk
	if createdDisk, err = c.i.GetDisk(project, zone, d.Name); err != nil {
		return err
	}
	*d = *createdDisk
	return nil
}

// CreateImage creates a GCE image.
// Only one of sourceDisk or sourceFile must be specified, sourceDisk is the
// url (full or partial) to the source disk, sourceFile is the full Google
// Cloud Storage URL where the disk image is stored.
func (c *client) CreateImage(project string, i *compute.Image) error {
	op, err := c.Retry(c.raw.Images.Insert(project, i).Do)
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, "", op.Name); err != nil {
		return err
	}

	var createdImage *compute.Image
	if createdImage, err = c.i.GetImage(project, i.Name); err != nil {
		return err
	}
	*i = *createdImage
	return nil
}

func (c *client) CreateInstance(project, zone string, i *compute.Instance) error {
	op, err := c.Retry(c.raw.Instances.Insert(project, zone, i).Do)
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, zone, op.Name); err != nil {
		return err
	}

	var createdInstance *compute.Instance
	if createdInstance, err = c.i.GetInstance(project, zone, i.Name); err != nil {
		return err
	}
	*i = *createdInstance
	return nil
}

// DeleteImage deletes a GCE image.
func (c *client) DeleteImage(project, name string) error {
	op, err := c.Retry(c.raw.Images.Delete(project, name).Do)
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, "", op.Name)
}

// DeleteDisk deletes a GCE persistent disk.
func (c *client) DeleteDisk(project, zone, name string) error {
	op, err := c.Retry(c.raw.Disks.Delete(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, zone, op.Name)
}

// DeleteInstance deletes a GCE instance.
func (c *client) DeleteInstance(project, zone, name string) error {
	op, err := c.Retry(c.raw.Instances.Delete(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, zone, op.Name)
}

// GetMachineType gets a GCE MachineType.
func (c *client) GetMachineType(project, zone, machineType string) (*compute.MachineType, error) {
	mt, err := c.raw.MachineTypes.Get(project, zone, machineType).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.MachineTypes.Get(project, zone, machineType).Do()
	}
	return mt, err
}

// ListMachineTypes gets a list of GCE MachineTypes.
func (c *client) ListMachineTypes(project, zone string) ([]*compute.MachineType, error) {
	var mts []*compute.MachineType
	var pt string
	for mtl, err := c.raw.MachineTypes.List(project, zone).PageToken(pt).Do(); ; mtl, err = c.raw.MachineTypes.List(project, zone).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			mtl, err = c.raw.MachineTypes.List(project, zone).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		mts = append(mts, mtl.Items...)

		if mtl.NextPageToken == "" {
			return mts, nil
		}
		pt = mtl.NextPageToken
	}
}

// GetProject gets a GCE Project.
func (c *client) GetProject(project string) (*compute.Project, error) {
	p, err := c.raw.Projects.Get(project).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Projects.Get(project).Do()
	}
	return p, err
}

// GetSerialPortOutput gets the serial port output of a GCE instance.
func (c *client) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	sp, err := c.raw.Instances.GetSerialPortOutput(project, zone, name).Start(start).Port(port).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Instances.GetSerialPortOutput(project, zone, name).Start(start).Port(port).Do()
	}
	return sp, err
}

// GetZone gets a GCE Zone.
func (c *client) GetZone(project, zone string) (*compute.Zone, error) {
	z, err := c.raw.Zones.Get(project, zone).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Zones.Get(project, zone).Do()
	}
	return z, err
}

// ListZones gets a list GCE Zones.
func (c *client) ListZones(project string) ([]*compute.Zone, error) {
	var zs []*compute.Zone
	var pt string
	for zl, err := c.raw.Zones.List(project).PageToken(pt).Do(); ; zl, err = c.raw.Zones.List(project).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			zl, err = c.raw.Zones.List(project).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		zs = append(zs, zl.Items...)

		if zl.NextPageToken == "" {
			return zs, nil
		}
		pt = zl.NextPageToken
	}
}

// GetInstance gets a GCE Instance.
func (c *client) GetInstance(project, zone, name string) (*compute.Instance, error) {
	i, err := c.raw.Instances.Get(project, zone, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Instances.Get(project, zone, name).Do()
	}
	return i, err
}

// ListInstances gets a list of GCE Instances.
func (c *client) ListInstances(project, zone string) ([]*compute.Instance, error) {
	var is []*compute.Instance
	var pt string
	for il, err := c.raw.Instances.List(project, zone).PageToken(pt).Do(); ; il, err = c.raw.Instances.List(project, zone).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			il, err = c.raw.Instances.List(project, zone).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		is = append(is, il.Items...)

		if il.NextPageToken == "" {
			return is, nil
		}
		pt = il.NextPageToken
	}
}

// GetDisk gets a GCE Disk.
func (c *client) GetDisk(project, zone, name string) (*compute.Disk, error) {
	d, err := c.raw.Disks.Get(project, zone, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Disks.Get(project, zone, name).Do()
	}
	return d, err
}

// ListDisks gets a list of GCE Disks.
func (c *client) ListDisks(project, zone string) ([]*compute.Disk, error) {
	var dl []*compute.Disk
	var pt string
	for d, err := c.raw.Disks.List(project, zone).PageToken(pt).Do(); ; d, err = c.raw.Disks.List(project, zone).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			d, err = c.raw.Disks.List(project, zone).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		dl = append(dl, d.Items...)

		if d.NextPageToken == "" {
			return dl, nil
		}
		pt = d.NextPageToken
	}
}

// GetImage gets a GCE Image.
func (c *client) GetImage(project, name string) (*compute.Image, error) {
	i, err := c.raw.Images.Get(project, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Images.Get(project, name).Do()
	}
	return i, err
}

// GetImageFromFamily gets a GCE Image from an image family.
func (c *client) GetImageFromFamily(project, family string) (*compute.Image, error) {
	i, err := c.raw.Images.GetFromFamily(project, family).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Images.GetFromFamily(project, family).Do()
	}
	return i, err
}

// ListImages gets a list of GCE Images.
func (c *client) ListImages(project string) ([]*compute.Image, error) {
	var is []*compute.Image
	var pt string
	for il, err := c.raw.Images.List(project).PageToken(pt).Do(); ; il, err = c.raw.Images.List(project).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			il, err = c.raw.Images.List(project).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		is = append(is, il.Items...)

		if il.NextPageToken == "" {
			return is, nil
		}
		pt = il.NextPageToken
	}
}

// GetNetwork gets a GCE Network.
func (c *client) GetNetwork(project, name string) (*compute.Network, error) {
	n, err := c.raw.Networks.Get(project, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Networks.Get(project, name).Do()
	}
	return n, err
}

// ListNetworks gets a list of GCE Networks.
func (c *client) ListNetworks(project string) ([]*compute.Network, error) {
	var nl []*compute.Network
	var pt string
	for n, err := c.raw.Networks.List(project).PageToken(pt).Do(); ; n, err = c.raw.Networks.List(project).PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			n, err = c.raw.Networks.List(project).PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		nl = append(nl, n.Items...)

		if n.NextPageToken == "" {
			return nl, nil
		}
		pt = n.NextPageToken
	}
}

// GetLicense gets a GCE License.
func (c *client) GetLicense(project, name string) (*compute.License, error) {
	l, err := c.raw.Licenses.Get(project, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Licenses.Get(project, name).Do()
	}
	return l, err
}

// InstanceStatus returns an instances Status.
func (c *client) InstanceStatus(project, zone, name string) (string, error) {
	is, err := c.raw.Instances.Get(project, zone, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		is, err = c.raw.Instances.Get(project, zone, name).Do()
	}

	if err != nil {
		return "", err
	}
	return is.Status, nil
}

// InstanceStopped checks if a GCE instance is in a 'TERMINATED' or 'STOPPED' state.
func (c *client) InstanceStopped(project, zone, name string) (bool, error) {
	status, err := c.i.InstanceStatus(project, zone, name)
	if err != nil {
		return false, err
	}
	switch status {
	case "PROVISIONING", "RUNNING", "STAGING", "STOPPING":
		return false, nil
	case "TERMINATED", "STOPPED":
		return true, nil
	default:
		return false, fmt.Errorf("unexpected instance status %q", status)
	}
}
