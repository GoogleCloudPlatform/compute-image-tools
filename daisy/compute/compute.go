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
	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

// Client is a client for interacting with Google Cloud Compute.
type Client interface {
	AttachDisk(project, zone, instance string, d *compute.AttachedDisk) error
	DetachDisk(project, zone, instance, disk string) error
	CreateDisk(project, zone string, d *compute.Disk) error
	CreateForwardingRule(project, region string, fr *compute.ForwardingRule) error
	CreateFirewallRule(project string, i *compute.Firewall) error
	CreateImage(project string, i *computeAlpha.Image) error
	CreateInstance(project, zone string, i *compute.Instance) error
	CreateNetwork(project string, n *compute.Network) error
	CreateSubnetwork(project, region string, n *compute.Subnetwork) error
	CreateTargetInstance(project, zone string, ti *compute.TargetInstance) error
	DeleteDisk(project, zone, name string) error
	DeleteForwardingRule(project, region, name string) error
	DeleteFirewallRule(project, name string) error
	DeleteImage(project, name string) error
	DeleteInstance(project, zone, name string) error
	StartInstance(project, zone, name string) error
	StopInstance(project, zone, name string) error
	DeleteNetwork(project, name string) error
	DeleteSubnetwork(project, region, name string) error
	DeleteTargetInstance(project, zone, name string) error
	DeprecateImage(project, name string, deprecationstatus *compute.DeprecationStatus) error
	GetMachineType(project, zone, machineType string) (*compute.MachineType, error)
	GetProject(project string) (*compute.Project, error)
	GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	GetZone(project, zone string) (*compute.Zone, error)
	GetInstance(project, zone, name string) (*compute.Instance, error)
	GetDisk(project, zone, name string) (*compute.Disk, error)
	GetForwardingRule(project, region, name string) (*compute.ForwardingRule, error)
	GetFirewallRule(project, name string) (*compute.Firewall, error)
	GetImage(project, name string) (*computeAlpha.Image, error)
	GetImageFromFamily(project, family string) (*computeAlpha.Image, error)
	GetLicense(project, name string) (*compute.License, error)
	GetNetwork(project, name string) (*compute.Network, error)
	GetSubnetwork(project, region, name string) (*compute.Subnetwork, error)
	GetTargetInstance(project, zone, name string) (*compute.TargetInstance, error)
	InstanceStatus(project, zone, name string) (string, error)
	InstanceStopped(project, zone, name string) (bool, error)
	ListMachineTypes(project, zone string, opts ...ListCallOption) ([]*compute.MachineType, error)
	ListZones(project string, opts ...ListCallOption) ([]*compute.Zone, error)
	ListRegions(project string, opts ...ListCallOption) ([]*compute.Region, error)
	ListInstances(project, zone string, opts ...ListCallOption) ([]*compute.Instance, error)
	ListDisks(project, zone string, opts ...ListCallOption) ([]*compute.Disk, error)
	ListForwardingRules(project, zone string, opts ...ListCallOption) ([]*compute.ForwardingRule, error)
	ListFirewallRules(project string, opts ...ListCallOption) ([]*compute.Firewall, error)
	ListImages(project string, opts ...ListCallOption) ([]*computeAlpha.Image, error)
	ListNetworks(project string, opts ...ListCallOption) ([]*compute.Network, error)
	ListSubnetworks(project, region string, opts ...ListCallOption) ([]*compute.Subnetwork, error)
	ListTargetInstances(project, zone string, opts ...ListCallOption) ([]*compute.TargetInstance, error)
	ResizeDisk(project, zone, disk string, drr *compute.DisksResizeRequest) error
	SetInstanceMetadata(project, zone, name string, md *compute.Metadata) error
	SetCommonInstanceMetadata(project string, md *compute.Metadata) error

	// Beta API calls
	GetGuestAttributes(project, zone, name, queryPath, variableKey string) (*computeBeta.GuestAttributes, error)

	Retry(f func(opts ...googleapi.CallOption) (*compute.Operation, error), opts ...googleapi.CallOption) (op *compute.Operation, err error)
	BasePath() string
}

// A ListCallOption is an option for a Google Compute API *ListCall.
type ListCallOption interface {
	listCallOptionApply(interface{}) interface{}
}

// OrderBy sets the optional parameter "orderBy": Sorts list results by a
// certain order. By default, results are returned in alphanumerical order
// based on the resource name.
type OrderBy string

func (o OrderBy) listCallOptionApply(i interface{}) interface{} {
	switch c := i.(type) {
	case *compute.FirewallsListCall:
		return c.OrderBy(string(o))
	case *compute.ImagesListCall:
		return c.OrderBy(string(o))
	case *compute.MachineTypesListCall:
		return c.OrderBy(string(o))
	case *compute.ZonesListCall:
		return c.OrderBy(string(o))
	case *compute.InstancesListCall:
		return c.OrderBy(string(o))
	case *compute.DisksListCall:
		return c.OrderBy(string(o))
	case *compute.NetworksListCall:
		return c.OrderBy(string(o))
	case *compute.SubnetworksListCall:
		return c.OrderBy(string(o))
	}
	return i
}

// Filter sets the optional parameter "filter": Sets a filter {expression} for
// filtering listed resources. Your {expression} must be in the format:
// field_name comparison_string literal_string.
type Filter string

func (o Filter) listCallOptionApply(i interface{}) interface{} {
	switch c := i.(type) {
	case *compute.FirewallsListCall:
		return c.Filter(string(o))
	case *compute.ImagesListCall:
		return c.Filter(string(o))
	case *compute.MachineTypesListCall:
		return c.Filter(string(o))
	case *compute.ZonesListCall:
		return c.Filter(string(o))
	case *compute.InstancesListCall:
		return c.Filter(string(o))
	case *compute.DisksListCall:
		return c.Filter(string(o))
	case *compute.NetworksListCall:
		return c.Filter(string(o))
	case *compute.SubnetworksListCall:
		return c.Filter(string(o))
	}
	return i
}

type clientImpl interface {
	Client
	zoneOperationsWait(project, zone, name string) error
	regionOperationsWait(project, region, name string) error
	globalOperationsWait(project, name string) error
}

type client struct {
	i        clientImpl
	hc       *http.Client
	raw      *compute.Service
	rawBeta  *computeBeta.Service
	rawAlpha *computeAlpha.Service
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
	o := []option.ClientOption{option.WithScopes(compute.ComputeScope, compute.DevstorageReadWriteScope)}
	opts = append(o, opts...)
	hc, ep, err := transport.NewHTTPClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP API client: %v", err)
	}
	rawService, err := compute.New(hc)
	if err != nil {
		return nil, fmt.Errorf("compute client: %v", err)
	}
	if ep != "" {
		rawService.BasePath = ep
	}
	rawBetaService, err := computeBeta.New(hc)
	if err != nil {
		return nil, fmt.Errorf("beta compute client: %v", err)
	}
	if ep != "" {
		rawBetaService.BasePath = ep
	}
	rawAlphaService, err := computeAlpha.New(hc)
	if err != nil {
		return nil, fmt.Errorf("alpha compute client: %v", err)
	}
	if ep != "" {
		rawAlphaService.BasePath = ep
	}

	c := &client{hc: hc, raw: rawService, rawBeta: rawBetaService, rawAlpha: rawAlphaService}
	c.i = c

	return c, nil
}

// BasePath returns the base path for this client.
func (c *client) BasePath() string {
	return c.raw.BasePath
}

type operationGetterFunc func() (*compute.Operation, error)

func (c *client) zoneOperationsWait(project, zone, name string) error {
	return c.operationsWaitHelper(project, name, func() (op *compute.Operation, err error) {
		op, err = c.Retry(c.raw.ZoneOperations.Get(project, zone, name).Do)
		if err != nil {
			err = fmt.Errorf("failed to get zone operation %s: %v", name, err)
		}
		return op, err
	})
}

func (c *client) regionOperationsWait(project, region, name string) error {
	return c.operationsWaitHelper(project, name, func() (op *compute.Operation, err error) {
		op, err = c.Retry(c.raw.RegionOperations.Get(project, region, name).Do)
		if err != nil {
			err = fmt.Errorf("failed to get region operation %s: %v", name, err)
		}
		return op, err
	})
}

func (c *client) globalOperationsWait(project, name string) error {
	return c.operationsWaitHelper(project, name, func() (op *compute.Operation, err error) {
		op, err = c.Retry(c.raw.GlobalOperations.Get(project, name).Do)
		if err != nil {
			err = fmt.Errorf("failed to get global operation %s: %v", name, err)
		}
		return op, err
	})
}

func (c *client) operationsWaitHelper(project, name string, getOperation operationGetterFunc) error {
	for {
		op, err := getOperation()
		if err != nil {
			return err
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

// Retry invokes the given function, retrying it multiple times if the HTTP
// status response indicates the request should be attempted again or the
// oauth Token is no longer valid.
func (c *client) RetryAlpha(f func(opts ...googleapi.CallOption) (*computeAlpha.Operation, error), opts ...googleapi.CallOption) (op *computeAlpha.Operation, err error) {
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

// AttachDisk attaches a GCE persistent disk to an instance.
func (c *client) AttachDisk(project, zone, instance string, d *compute.AttachedDisk) error {
	op, err := c.Retry(c.raw.Instances.AttachDisk(project, zone, instance, d).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// DetachDisk detaches a GCE persistent disk to an instance.
func (c *client) DetachDisk(project, zone, instance, disk string) error {
	op, err := c.Retry(c.raw.Instances.DetachDisk(project, zone, instance, disk).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// CreateDisk creates a GCE persistent disk.
func (c *client) CreateDisk(project, zone string, d *compute.Disk) error {
	op, err := c.Retry(c.raw.Disks.Insert(project, zone, d).Do)
	if err != nil {
		return err
	}

	if err := c.i.zoneOperationsWait(project, zone, op.Name); err != nil {
		return err
	}

	var createdDisk *compute.Disk
	if createdDisk, err = c.i.GetDisk(project, zone, d.Name); err != nil {
		return err
	}
	*d = *createdDisk
	return nil
}

// CreateForwardingRule creates a GCE forwarding rule.
func (c *client) CreateForwardingRule(project, region string, fr *compute.ForwardingRule) error {
	op, err := c.Retry(c.raw.ForwardingRules.Insert(project, region, fr).Do)
	if err != nil {
		return err
	}

	if err := c.i.regionOperationsWait(project, region, op.Name); err != nil {
		return err
	}

	var createdForwardingRule *compute.ForwardingRule
	if createdForwardingRule, err = c.i.GetForwardingRule(project, region, fr.Name); err != nil {
		return err
	}
	*fr = *createdForwardingRule
	return nil
}

func (c *client) CreateFirewallRule(project string, i *compute.Firewall) error {
	op, err := c.Retry(c.raw.Firewalls.Insert(project, i).Do)
	if err != nil {
		return err
	}

	if err := c.i.globalOperationsWait(project, op.Name); err != nil {
		return err
	}

	var createdFirewallRule *compute.Firewall
	if createdFirewallRule, err = c.i.GetFirewallRule(project, i.Name); err != nil {
		return err
	}
	*i = *createdFirewallRule
	return nil
}

// CreateImage creates a GCE image.
// Only one of sourceDisk or sourceFile must be specified, sourceDisk is the
// url (full or partial) to the source disk, sourceFile is the full Google
// Cloud Storage URL where the disk image is stored.
func (c *client) CreateImage(project string, i *computeAlpha.Image) error {
	op, err := c.RetryAlpha(c.rawAlpha.Images.Insert(project, i).Do)
	if err != nil {
		return err
	}

	if err := c.i.globalOperationsWait(project, op.Name); err != nil {
		return err
	}

	var createdImage *computeAlpha.Image
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

	if err := c.i.zoneOperationsWait(project, zone, op.Name); err != nil {
		return err
	}

	var createdInstance *compute.Instance
	if createdInstance, err = c.i.GetInstance(project, zone, i.Name); err != nil {
		return err
	}
	*i = *createdInstance
	return nil
}

func (c *client) CreateNetwork(project string, n *compute.Network) error {
	op, err := c.Retry(c.raw.Networks.Insert(project, n).Do)
	if err != nil {
		return err
	}

	if err := c.i.globalOperationsWait(project, op.Name); err != nil {
		return err
	}

	var createdNetwork *compute.Network
	if createdNetwork, err = c.i.GetNetwork(project, n.Name); err != nil {
		return err
	}
	*n = *createdNetwork
	return nil
}

func (c *client) CreateSubnetwork(project, region string, n *compute.Subnetwork) error {
	op, err := c.Retry(c.raw.Subnetworks.Insert(project, region, n).Do)
	if err != nil {
		return err
	}

	if err := c.i.regionOperationsWait(project, region, op.Name); err != nil {
		return err
	}

	var createdSubnetwork *compute.Subnetwork
	if createdSubnetwork, err = c.i.GetSubnetwork(project, region, n.Name); err != nil {
		return err
	}
	*n = *createdSubnetwork
	return nil
}

// CreateTargetInstance creates a GCE Target Instance, which can be used as
// target on ForwardingRule
func (c *client) CreateTargetInstance(project, zone string, ti *compute.TargetInstance) error {
	op, err := c.Retry(c.raw.TargetInstances.Insert(project, zone, ti).Do)
	if err != nil {
		return err
	}

	if err := c.i.zoneOperationsWait(project, zone, op.Name); err != nil {
		return err
	}

	var createdTargetInstance *compute.TargetInstance
	if createdTargetInstance, err = c.i.GetTargetInstance(project, zone, ti.Name); err != nil {
		return err
	}
	*ti = *createdTargetInstance
	return nil
}

// DeleteFirewallRule deletes a GCE FirewallRule.
func (c *client) DeleteFirewallRule(project, name string) error {
	op, err := c.Retry(c.raw.Firewalls.Delete(project, name).Do)
	if err != nil {
		return err
	}

	return c.i.globalOperationsWait(project, op.Name)
}

// DeleteImage deletes a GCE image.
func (c *client) DeleteImage(project, name string) error {
	op, err := c.Retry(c.raw.Images.Delete(project, name).Do)
	if err != nil {
		return err
	}

	return c.i.globalOperationsWait(project, op.Name)
}

// DeleteDisk deletes a GCE persistent disk.
func (c *client) DeleteDisk(project, zone, name string) error {
	op, err := c.Retry(c.raw.Disks.Delete(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// DeleteForwardingRule deletes a GCE ForwardingRule.
func (c *client) DeleteForwardingRule(project, region, name string) error {
	op, err := c.Retry(c.raw.ForwardingRules.Delete(project, region, name).Do)
	if err != nil {
		return err
	}

	return c.i.regionOperationsWait(project, region, op.Name)
}

// DeleteInstance deletes a GCE instance.
func (c *client) DeleteInstance(project, zone, name string) error {
	op, err := c.Retry(c.raw.Instances.Delete(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// StartInstance starts a GCE instance.
func (c *client) StartInstance(project, zone, name string) error {
	op, err := c.Retry(c.raw.Instances.Start(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// StopInstance stops a GCE instance.
func (c *client) StopInstance(project, zone, name string) error {
	op, err := c.Retry(c.raw.Instances.Stop(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// DeleteNetwork deletes a GCE network.
func (c *client) DeleteNetwork(project, name string) error {
	op, err := c.Retry(c.raw.Networks.Delete(project, name).Do)
	if err != nil {
		return err
	}

	return c.i.globalOperationsWait(project, op.Name)
}

// DeleteSubnetwork deletes a GCE subnetwork.
func (c *client) DeleteSubnetwork(project, region, name string) error {
	op, err := c.Retry(c.raw.Subnetworks.Delete(project, region, name).Do)
	if err != nil {
		return err
	}

	return c.i.regionOperationsWait(project, region, op.Name)
}

// DeleteTargetInstance deletes a GCE TargetInstance.
func (c *client) DeleteTargetInstance(project, zone, name string) error {
	op, err := c.Retry(c.raw.TargetInstances.Delete(project, zone, name).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// DeprecateImage sets deprecation status on a GCE image.
func (c *client) DeprecateImage(project, name string, deprecationstatus *compute.DeprecationStatus) error {
	op, err := c.Retry(c.raw.Images.Deprecate(project, name, deprecationstatus).Do)
	if err != nil {
		return err
	}

	return c.i.globalOperationsWait(project, op.Name)
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
func (c *client) ListMachineTypes(project, zone string, opts ...ListCallOption) ([]*compute.MachineType, error) {
	var mts []*compute.MachineType
	var pt string
	call := c.raw.MachineTypes.List(project, zone)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.MachineTypesListCall)
	}
	for mtl, err := call.PageToken(pt).Do(); ; mtl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			mtl, err = call.PageToken(pt).Do()
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
func (c *client) ListZones(project string, opts ...ListCallOption) ([]*compute.Zone, error) {
	var zs []*compute.Zone
	var pt string
	call := c.raw.Zones.List(project)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.ZonesListCall)
	}
	for zl, err := call.PageToken(pt).Do(); ; zl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			zl, err = call.PageToken(pt).Do()
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

// ListRegions gets a list GCE Regions.
func (c *client) ListRegions(project string, opts ...ListCallOption) ([]*compute.Region, error) {
	var rs []*compute.Region
	var pt string
	call := c.raw.Regions.List(project)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.RegionsListCall)
	}
	for rl, err := call.PageToken(pt).Do(); ; rl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			rl, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		rs = append(rs, rl.Items...)

		if rl.NextPageToken == "" {
			return rs, nil
		}
		pt = rl.NextPageToken
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
func (c *client) ListInstances(project, zone string, opts ...ListCallOption) ([]*compute.Instance, error) {
	var is []*compute.Instance
	var pt string
	call := c.raw.Instances.List(project, zone)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.InstancesListCall)
	}
	for il, err := call.PageToken(pt).Do(); ; il, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			il, err = call.PageToken(pt).Do()
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
func (c *client) ListDisks(project, zone string, opts ...ListCallOption) ([]*compute.Disk, error) {
	var ds []*compute.Disk
	var pt string
	call := c.raw.Disks.List(project, zone)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.DisksListCall)
	}
	for dl, err := call.PageToken(pt).Do(); ; dl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			dl, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		ds = append(ds, dl.Items...)

		if dl.NextPageToken == "" {
			return ds, nil
		}
		pt = dl.NextPageToken
	}
}

// GetForwardingRule gets a GCE ForwardingRule.
func (c *client) GetForwardingRule(project, region, name string) (*compute.ForwardingRule, error) {
	n, err := c.raw.ForwardingRules.Get(project, region, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.ForwardingRules.Get(project, region, name).Do()
	}
	return n, err
}

// ListForwardingRules gets a list of GCE ForwardingRules.
func (c *client) ListForwardingRules(project, region string, opts ...ListCallOption) ([]*compute.ForwardingRule, error) {
	var frs []*compute.ForwardingRule
	var pt string
	call := c.raw.ForwardingRules.List(project, region)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.ForwardingRulesListCall)
	}
	for frl, err := call.PageToken(pt).Do(); ; frl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			frl, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		frs = append(frs, frl.Items...)

		if frl.NextPageToken == "" {
			return frs, nil
		}
		pt = frl.NextPageToken
	}
}

// GetFirewallRule gets a GCE FirewallRule.
func (c *client) GetFirewallRule(project, name string) (*compute.Firewall, error) {
	i, err := c.raw.Firewalls.Get(project, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Firewalls.Get(project, name).Do()
	}
	return i, err
}

// ListFirewallRules gets a list of GCE FirewallRules.
func (c *client) ListFirewallRules(project string, opts ...ListCallOption) ([]*compute.Firewall, error) {
	var is []*compute.Firewall
	var pt string
	call := c.raw.Firewalls.List(project)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.FirewallsListCall)
	}
	for il, err := call.PageToken(pt).Do(); ; il, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			il, err = call.PageToken(pt).Do()
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

// GetImage gets a GCE Image.
func (c *client) GetImage(project, name string) (*computeAlpha.Image, error) {
	i, err := c.rawAlpha.Images.Get(project, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.rawAlpha.Images.Get(project, name).Do()
	}
	return i, err
}

// GetImageFromFamily gets a GCE Image from an image family.
func (c *client) GetImageFromFamily(project, family string) (*computeAlpha.Image, error) {
	i, err := c.rawAlpha.Images.GetFromFamily(project, family).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.rawAlpha.Images.GetFromFamily(project, family).Do()
	}
	return i, err
}

// ListImages gets a list of GCE Images.
func (c *client) ListImages(project string, opts ...ListCallOption) ([]*computeAlpha.Image, error) {
	var is []*computeAlpha.Image
	var pt string
	call := c.rawAlpha.Images.List(project)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*computeAlpha.ImagesListCall)
	}
	for il, err := call.PageToken(pt).Do(); ; il, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			il, err = call.PageToken(pt).Do()
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
func (c *client) ListNetworks(project string, opts ...ListCallOption) ([]*compute.Network, error) {
	var ns []*compute.Network
	var pt string
	call := c.raw.Networks.List(project)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.NetworksListCall)
	}
	for nl, err := call.PageToken(pt).Do(); ; nl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			nl, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		ns = append(ns, nl.Items...)

		if nl.NextPageToken == "" {
			return ns, nil
		}
		pt = nl.NextPageToken
	}
}

// GetSubnetwork gets a GCE subnetwork.
func (c *client) GetSubnetwork(project, region, name string) (*compute.Subnetwork, error) {
	n, err := c.raw.Subnetworks.Get(project, region, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.Subnetworks.Get(project, region, name).Do()
	}
	return n, err
}

// ListSubnetworks gets a list of GCE subnetworks.
func (c *client) ListSubnetworks(project, region string, opts ...ListCallOption) ([]*compute.Subnetwork, error) {
	var ns []*compute.Subnetwork
	var pt string
	call := c.raw.Subnetworks.List(project, region)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.SubnetworksListCall)
	}
	for nl, err := call.PageToken(pt).Do(); ; nl, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			nl, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		ns = append(ns, nl.Items...)

		if nl.NextPageToken == "" {
			return ns, nil
		}
		pt = nl.NextPageToken
	}
}

// GetTargetInstance gets a GCE TargetInstance.
func (c *client) GetTargetInstance(project, zone, name string) (*compute.TargetInstance, error) {
	n, err := c.raw.TargetInstances.Get(project, zone, name).Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return c.raw.TargetInstances.Get(project, zone, name).Do()
	}
	return n, err
}

// ListTargetInstances gets a list of GCE TargetInstances.
func (c *client) ListTargetInstances(project, zone string, opts ...ListCallOption) ([]*compute.TargetInstance, error) {
	var tis []*compute.TargetInstance
	var pt string
	call := c.raw.TargetInstances.List(project, zone)
	for _, opt := range opts {
		call = opt.listCallOptionApply(call).(*compute.TargetInstancesListCall)
	}
	for til, err := call.PageToken(pt).Do(); ; til, err = call.PageToken(pt).Do() {
		if shouldRetryWithWait(c.hc.Transport, err, 2) {
			til, err = call.PageToken(pt).Do()
		}
		if err != nil {
			return nil, err
		}
		tis = append(tis, til.Items...)

		if til.NextPageToken == "" {
			return tis, nil
		}
		pt = til.NextPageToken
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

// ResizeDisk resizes a GCE persistent disk. You can only increase the size of the disk.
func (c *client) ResizeDisk(project, zone, disk string, drr *compute.DisksResizeRequest) error {
	op, err := c.Retry(c.raw.Disks.Resize(project, zone, disk, drr).Do)
	if err != nil {
		return err
	}

	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// SetInstanceMetadata sets an instances metadata.
func (c *client) SetInstanceMetadata(project, zone, name string, md *compute.Metadata) error {
	op, err := c.Retry(c.raw.Instances.SetMetadata(project, zone, name, md).Do)
	if err != nil {
		return err
	}
	return c.i.zoneOperationsWait(project, zone, op.Name)
}

// SetCommonInstanceMetadata sets an instances metadata.
func (c *client) SetCommonInstanceMetadata(project string, md *compute.Metadata) error {
	op, err := c.Retry(c.raw.Projects.SetCommonInstanceMetadata(project, md).Do)
	if err != nil {
		return err
	}

	return c.i.globalOperationsWait(project, op.Name)
}

// GetGuestAttributes gets a Guest Attributes.
func (c *client) GetGuestAttributes(project, zone, name, queryPath, variableKey string) (*computeBeta.GuestAttributes, error) {
	call := c.rawBeta.Instances.GetGuestAttributes(project, zone, name)
	if queryPath != "" {
		call = call.QueryPath(queryPath)
	}
	if variableKey != "" {
		call = call.VariableKey(variableKey)
	}
	a, err := call.Do()
	if shouldRetryWithWait(c.hc.Transport, err, 2) {
		return call.Do()
	}
	return a, err
}
