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
	"net/http"
	"time"

	compute "google.golang.org/api/compute/v1"
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
	GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	InstanceStatus(project, zone, name string) (string, error)
	InstanceStopped(project, zone, name string) (bool, error)
	NewInstance(name, project, zone, machineType string) (*Instance, error)
	WaitForInstanceStopped(project, zone, name string, interval time.Duration) error
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
			op, err = c.raw.ZoneOperations.Get(project, zone, name).Do()
			if err != nil {
				return fmt.Errorf("Failed to get operation %s: %v", name, err)
			}
		} else {
			op, err = c.raw.GlobalOperations.Get(project, name).Do()
			if err != nil {
				return fmt.Errorf("Failed to get operation %s: %v", name, err)
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

// CreateDisk creates a GCE persistent disk.
func (c *client) CreateDisk(project, zone string, d *compute.Disk) error {
	resp, err := c.raw.Disks.Insert(project, zone, d).Do()
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, zone, resp.Name); err != nil {
		return err
	}

	var createdDisk *compute.Disk
	if createdDisk, err = c.raw.Disks.Get(project, zone, d.Name).Do(); err != nil {
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
	resp, err := c.raw.Images.Insert(project, i).Do()
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, "", resp.Name); err != nil {
		return err
	}

	var createdImage *compute.Image
	if createdImage, err = c.raw.Images.Get(project, i.Name).Do(); err != nil {
		return err
	}
	*i = *createdImage
	return nil
}

func (c *client) CreateInstance(project, zone string, i *compute.Instance) error {
	resp, err := c.raw.Instances.Insert(project, zone, i).Do()
	if err != nil {
		return err
	}

	if err := c.i.operationsWait(project, zone, resp.Name); err != nil {
		return err
	}

	var createdInstance *compute.Instance
	if createdInstance, err = c.raw.Instances.Get(project, zone, i.Name).Do(); err != nil {
		return err
	}
	*i = *createdInstance
	return nil
}

// DeleteImage deletes a GCE image.
func (c *client) DeleteImage(project, name string) error {
	resp, err := c.raw.Images.Delete(project, name).Do()
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, "", resp.Name)
}

// DeleteDisk deletes a GCE persistent disk.
func (c *client) DeleteDisk(project, zone, name string) error {
	resp, err := c.raw.Disks.Delete(project, zone, name).Do()
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, zone, resp.Name)
}

// DeleteInstance deletes a GCE instance.
func (c *client) DeleteInstance(project, zone, name string) error {
	resp, err := c.raw.Instances.Delete(project, zone, name).Do()
	if err != nil {
		return err
	}

	return c.i.operationsWait(project, zone, resp.Name)
}

// GetSerialPortOutput gets the serial port output of a GCE instance.
func (c *client) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	return c.raw.Instances.GetSerialPortOutput(project, zone, name).Start(start).Port(port).Do()
}

// InstanceStatus returns an instances Status.
func (c *client) InstanceStatus(project, zone, name string) (string, error) {
	inst, err := c.raw.Instances.Get(project, zone, name).Do()
	if err != nil {
		return "", err
	}
	return inst.Status, nil
}

// InstanceStopped checks if a GCE instance is in a 'TERMINATED' state.
func (c *client) InstanceStopped(project, zone, name string) (bool, error) {
	status, err := c.i.InstanceStatus(project, zone, name)
	if err != nil {
		return false, err
	}
	switch status {
	case "PROVISIONING", "RUNNING", "STAGING", "STOPPING":
		return false, nil
	case "TERMINATED":
		return true, nil
	default:
		return false, fmt.Errorf("unexpected instance status %q", status)
	}
}

// WaitForInstanceStopped waits a GCE instance to enter 'TERMINATED' state.
func (c *client) WaitForInstanceStopped(project, zone, instance string, interval time.Duration) error {
	for {
		stopped, err := c.i.InstanceStopped(project, zone, instance)
		if err != nil {
			return err
		}
		switch stopped {
		case true:
			return nil
		case false:
			time.Sleep(interval)
		}
	}
}

// Instance is the definition of a GCE instance.
type Instance struct {
	client            *client
	name              string
	zone              string
	project           string
	machineType       string
	disks             []*compute.AttachedDisk
	networkInterfaces []*compute.NetworkInterface
	metadata          *compute.Metadata

	// Optional description of the instance.
	Description string
	// Optional scopes of the instance.
	Scopes []string
}

func (i *Instance) checkMachineType() error {
	if i.machineType == "" {
		i.machineType = "n1-standard-1"
		return nil
	}
	list, err := i.client.raw.MachineTypes.List(i.project, i.zone).Do()
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if item.Name == i.machineType {
			return nil
		}
	}
	return fmt.Errorf("unknown machine type: %s, project: %s, zone: %s", i.machineType, i.project, i.zone)
}

// AddPD adds an additional disk from image to the instance.
func (i *Instance) AddPD(name, source, mode string, autoDelete, boot bool) {
	i.disks = append(i.disks, &compute.AttachedDisk{
		AutoDelete: autoDelete,
		Boot:       boot,
		DeviceName: name,
		Mode:       mode,
		Type:       "PERSISTENT",
		Source:     source,
	})
}

// AddNetworkInterface adds the network interface to the instance.
func (i *Instance) AddNetworkInterface(network string) {
	i.networkInterfaces = append(i.networkInterfaces, &compute.NetworkInterface{
		AccessConfigs: []*compute.AccessConfig{
			{
				Type: "ONE_TO_ONE_NAT",
			},
		},
		Network: network,
	})
}

// AddNetworkInterfaceWithSubnetwork adds the network interface to the instance.
func (i *Instance) AddNetworkInterfaceWithSubnetwork(network, subnetwork string) {
	i.networkInterfaces = append(i.networkInterfaces, &compute.NetworkInterface{
		AccessConfigs: []*compute.AccessConfig{
			{
				Type: "ONE_TO_ONE_NAT",
			},
		},
		Network:    network,
		Subnetwork: subnetwork,
	})
}

// Insert inserts a new instance on GCE.
func (i *Instance) Insert() (*compute.Instance, error) {
	prefix := "https://www.googleapis.com/compute/v1/projects/" + i.project
	machineType := prefix + "/zones/" + i.zone + "/machineTypes/" + i.machineType

	scopes := []string{"https://www.googleapis.com/auth/devstorage.read_only"}
	if len(i.Scopes) > 0 {
		scopes = i.Scopes
	}
	serviceAccounts := []*compute.ServiceAccount{
		{
			Email:  "default",
			Scopes: scopes,
		},
	}

	instance := &compute.Instance{
		Name:              i.name,
		Description:       i.Description,
		MachineType:       machineType,
		Disks:             i.disks,
		Metadata:          i.metadata,
		NetworkInterfaces: i.networkInterfaces,
		ServiceAccounts:   serviceAccounts,
	}

	resp, err := i.client.raw.Instances.Insert(i.project, i.zone, instance).Do()
	if err != nil {
		return nil, fmt.Errorf("Failed to create instance: %v", err)
	}

	if err := i.client.operationsWait(i.project, i.zone, resp.Name); err != nil {
		return nil, err
	}
	return i.client.raw.Instances.Get(i.project, i.zone, i.name).Do()
}

// NewInstance creates a new Instance struct.
func (c *client) NewInstance(name, project, zone, machineType string) (*Instance, error) {
	instance := &Instance{
		client:      c,
		name:        name,
		zone:        zone,
		project:     project,
		machineType: machineType,
	}

	if err := instance.checkMachineType(); err != nil {
		return nil, err
	}

	return instance, nil
}

// AddMetadata adds metadata to the instance.
func (i *Instance) AddMetadata(metadata map[string]string) {
	var md []*compute.MetadataItems
	for k, v := range metadata {
		newV := v
		md = append(md, &compute.MetadataItems{Key: k, Value: &newV})
	}
	if i.metadata == nil {
		i.metadata = &compute.Metadata{Items: md}
	} else {
		i.metadata.Items = append(md, i.metadata.Items...)
	}
}
