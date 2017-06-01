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

type Client interface {
	CreateDisk(project, zone string, d *compute.Disk) error
	CreateImage(project string, i *compute.Image) error
	DeleteDisk(project, zone, name string) error
	DeleteImage(project, name string) error
	DeleteInstance(project, zone, name string) error
	GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	InstanceStatus(project, zone, name string) (string, error)
	InstanceStopped(project, zone, name string) (bool, error)
  WaitForInstanceStopped(project, zone, name string, interval time.Duration) error
}

// Client is a client for interacting with Google Cloud Compute.
type realClient struct {
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
	c := &realClient{
		hc:  hc,
		raw: rawService,
	}

	return c, nil
}

func (c *realClient) operationsWait(project, zone, name string) error {
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
func (c *realClient) CreateDisk(project, zone string, d *compute.Disk) error {
	fmt.Println("here1")
	resp, err := c.raw.Disks.Insert(project, zone, d).Do()
	if err != nil {
		return err
	}

	fmt.Println("here2")
	if err := c.operationsWait(project, zone, resp.Name); err != nil {
		return err
	}
	fmt.Println("here3")

	var createdDisk *compute.Disk
	if createdDisk, err = c.raw.Disks.Get(project, zone, d.Name).Do(); err != nil {
		return err
	}
	fmt.Println("here4")
	*d = *createdDisk
	return nil
}

// CreateImage creates a GCE image.
// Only one of sourceDisk or sourceFile must be specified, sourceDisk is the
// url (full or partial) to the source disk, sourceFile is the full Google
// Cloud Storage URL where the disk image is stored.
func (c *realClient) CreateImage(project string, i *compute.Image) error {
	resp, err := c.raw.Images.Insert(project, i).Do()
	if err != nil {
		return err
	}

	if err := c.operationsWait(project, "", resp.Name); err != nil {
		return err
	}

	var createdImage *compute.Image
	if createdImage, err = c.raw.Images.Get(project, i.Name).Do(); err != nil {
		return err
	}
	*i = *createdImage
	return nil
}

// DeleteImage deletes a GCE image.
func (c *realClient) DeleteImage(project, image string) error {
	resp, err := c.raw.Images.Delete(project, image).Do()
	if err != nil {
		return err
	}

	return c.operationsWait(project, "", resp.Name)
}

// DeleteDisk deletes a GCE persistent disk.
func (c *realClient) DeleteDisk(project, zone, disk string) error {
	resp, err := c.raw.Disks.Delete(project, zone, disk).Do()
	if err != nil {
		return err
	}

	return c.operationsWait(project, zone, resp.Name)
}

// DeleteInstance deletes a GCE instance.
func (c *realClient) DeleteInstance(project, zone, instance string) error {
	resp, err := c.raw.Instances.Delete(project, zone, instance).Do()
	if err != nil {
		return err
	}

	return c.operationsWait(project, zone, resp.Name)
}

// GetSerialPortOutput gets the serial port output of a GCE instance.
func (c *realClient) GetSerialPortOutput(project, zone, instance string, port, start int64) (*compute.SerialPortOutput, error) {
	return c.raw.Instances.GetSerialPortOutput(project, zone, instance).Start(start).Port(port).Do()
}

// InstanceStatus returns an instances Status.
func (c *realClient) InstanceStatus(project, zone, instance string) (string, error) {
	inst, err := c.raw.Instances.Get(project, zone, instance).Do()
	if err != nil {
		return "", err
	}
	return inst.Status, nil
}

// InstanceStopped checks if a GCE instance is in a 'TERMINATED' state.
func (c *realClient) InstanceStopped(project, zone, instance string) (bool, error) {
	status, err := c.InstanceStatus(project, zone, instance)
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
func (c *realClient) WaitForInstanceStopped(project, zone, instance string, interval time.Duration) error {
	for {
		stopped, err := c.InstanceStopped(project, zone, instance)
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
	client            *realClient
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
func (c *realClient) NewInstance(name, project, zone, machineType string) (*Instance, error) {
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
