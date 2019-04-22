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

	computeAlpha "google.golang.org/api/compute/v0.alpha"
	computeBeta "google.golang.org/api/compute/v0.beta"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
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

	AttachDiskFn                func(project, zone, instance string, d *compute.AttachedDisk) error
	DetachDiskFn                func(project, zone, instance, disk string) error
	CreateDiskFn                func(project, zone string, d *compute.Disk) error
	CreateForwardingRuleFn      func(project, region string, fr *compute.ForwardingRule) error
	CreateFirewallRuleFn        func(project string, i *compute.Firewall) error
	CreateImageFn               func(project string, i *computeAlpha.Image) error
	CreateInstanceFn            func(project, zone string, i *compute.Instance) error
	CreateNetworkFn             func(project string, n *compute.Network) error
	CreateSubnetworkFn          func(project, region string, n *compute.Subnetwork) error
	CreateTargetInstanceFn      func(project, zone string, ti *compute.TargetInstance) error
	StartInstanceFn             func(project, zone, name string) error
	StopInstanceFn              func(project, zone, name string) error
	DeleteDiskFn                func(project, zone, name string) error
	DeleteForwardingRuleFn      func(project, region, name string) error
	DeleteFirewallRuleFn        func(project, name string) error
	DeleteImageFn               func(project, name string) error
	DeleteInstanceFn            func(project, zone, name string) error
	DeleteNetworkFn             func(project, name string) error
	DeleteSubnetworkFn          func(project, region, name string) error
	DeleteTargetInstanceFn      func(project, zone, name string) error
	DeprecateImageFn            func(project, name string, deprecationstatus *compute.DeprecationStatus) error
	GetMachineTypeFn            func(project, zone, machineType string) (*compute.MachineType, error)
	ListMachineTypesFn          func(project, zone string, opts ...ListCallOption) ([]*compute.MachineType, error)
	GetProjectFn                func(project string) (*compute.Project, error)
	GetSerialPortOutputFn       func(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error)
	GetZoneFn                   func(project, zone string) (*compute.Zone, error)
	ListZonesFn                 func(project string, opts ...ListCallOption) ([]*compute.Zone, error)
	GetInstanceFn               func(project, zone, name string) (*compute.Instance, error)
	ListInstancesFn             func(project, zone string, opts ...ListCallOption) ([]*compute.Instance, error)
	GetDiskFn                   func(project, zone, name string) (*compute.Disk, error)
	ListDisksFn                 func(project, zone string, opts ...ListCallOption) ([]*compute.Disk, error)
	GetForwardingRuleFn         func(project, region, name string) (*compute.ForwardingRule, error)
	ListForwardingRulesFn       func(project, region string, opts ...ListCallOption) ([]*compute.ForwardingRule, error)
	GetFirewallRuleFn           func(project, name string) (*compute.Firewall, error)
	ListFirewallRulesFn         func(project string, opts ...ListCallOption) ([]*compute.Firewall, error)
	GetImageFn                  func(project, name string) (*computeAlpha.Image, error)
	GetImageFromFamilyFn        func(project, family string) (*computeAlpha.Image, error)
	ListImagesFn                func(project string, opts ...ListCallOption) ([]*computeAlpha.Image, error)
	GetLicenseFn                func(project, name string) (*compute.License, error)
	GetNetworkFn                func(project, name string) (*compute.Network, error)
	ListNetworksFn              func(project string, opts ...ListCallOption) ([]*compute.Network, error)
	GetSubnetworkFn             func(project, region, name string) (*compute.Subnetwork, error)
	ListSubnetworksFn           func(project, region string, opts ...ListCallOption) ([]*compute.Subnetwork, error)
	GetTargetInstanceFn         func(project, zone, name string) (*compute.TargetInstance, error)
	ListTargetInstancesFn       func(project, zone string, opts ...ListCallOption) ([]*compute.TargetInstance, error)
	InstanceStatusFn            func(project, zone, name string) (string, error)
	InstanceStoppedFn           func(project, zone, name string) (bool, error)
	ResizeDiskFn                func(project, zone, disk string, drr *compute.DisksResizeRequest) error
	SetInstanceMetadataFn       func(project, zone, name string, md *compute.Metadata) error
	SetCommonInstanceMetadataFn func(project string, md *compute.Metadata) error
	RetryFn                     func(f func(opts ...googleapi.CallOption) (*compute.Operation, error), opts ...googleapi.CallOption) (op *compute.Operation, err error)

	// Beta API calls
	GetGuestAttributesFn func(project, zone, name, queryPath, variableKey string) (*computeBeta.GuestAttributes, error)

	zoneOperationsWaitFn   func(project, zone, name string) error
	regionOperationsWaitFn func(project, region, name string) error
	globalOperationsWaitFn func(project, name string) error
}

// Retry uses the override method RetryFn or the real implementation.
func (c *TestClient) Retry(f func(opts ...googleapi.CallOption) (*compute.Operation, error), opts ...googleapi.CallOption) (op *compute.Operation, err error) {
	if c.RetryFn != nil {
		return c.RetryFn(f, opts...)
	}
	return c.client.Retry(f, opts...)
}

// AttachDisk uses the override method AttachDiskFn or the real implementation.
func (c *TestClient) AttachDisk(project, zone, instance string, ad *compute.AttachedDisk) error {
	if c.AttachDiskFn != nil {
		return c.AttachDiskFn(project, zone, instance, ad)
	}
	return c.client.AttachDisk(project, zone, instance, ad)
}

// DetachDisk uses the override method DetachDiskFn or the real implementation.
func (c *TestClient) DetachDisk(project, zone, instance, disk string) error {
	if c.DetachDiskFn != nil {
		return c.DetachDiskFn(project, zone, instance, disk)
	}
	return c.client.DetachDisk(project, zone, instance, disk)
}

// CreateDisk uses the override method CreateDiskFn or the real implementation.
func (c *TestClient) CreateDisk(project, zone string, d *compute.Disk) error {
	if c.CreateDiskFn != nil {
		return c.CreateDiskFn(project, zone, d)
	}
	return c.client.CreateDisk(project, zone, d)
}

// CreateForwardingRule uses the override method CreateForwardingRuleFn or the real implementation.
func (c *TestClient) CreateForwardingRule(project, region string, fr *compute.ForwardingRule) error {
	if c.CreateForwardingRuleFn != nil {
		return c.CreateForwardingRuleFn(project, region, fr)
	}
	return c.client.CreateForwardingRule(project, region, fr)
}

// CreateFirewallRule uses the override method CreateFirewallRuleFn or the real implementation.
func (c *TestClient) CreateFirewallRule(project string, i *compute.Firewall) error {
	if c.CreateFirewallRuleFn != nil {
		return c.CreateFirewallRuleFn(project, i)
	}
	return c.client.CreateFirewallRule(project, i)
}

// CreateImage uses the override method CreateImageFn or the real implementation.
func (c *TestClient) CreateImage(project string, i *computeAlpha.Image) error {
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

// CreateNetwork uses the override method CreateNetworkFn or the real implementation.
func (c *TestClient) CreateNetwork(project string, n *compute.Network) error {
	if c.CreateNetworkFn != nil {
		return c.CreateNetworkFn(project, n)
	}
	return c.client.CreateNetwork(project, n)
}

// CreateSubnetwork uses the override method CreateSubnetworkFn or the real implementation.
func (c *TestClient) CreateSubnetwork(project, region string, n *compute.Subnetwork) error {
	if c.CreateSubnetworkFn != nil {
		return c.CreateSubnetworkFn(project, region, n)
	}
	return c.client.CreateSubnetwork(project, region, n)
}

// CreateTargetInstance uses the override method CreateTargetInstanceFn or the real implementation.
func (c *TestClient) CreateTargetInstance(project, zone string, ti *compute.TargetInstance) error {
	if c.CreateTargetInstanceFn != nil {
		return c.CreateTargetInstanceFn(project, zone, ti)
	}
	return c.client.CreateTargetInstance(project, zone, ti)
}

// StartInstance uses the override method StartInstanceFn or the real implementation.
func (c *TestClient) StartInstance(project, zone, name string) error {
	if c.StartInstanceFn != nil {
		return c.StartInstanceFn(project, zone, name)
	}
	return c.client.StartInstance(project, zone, name)
}

// StopInstance uses the override method StopInstanceFn or the real implementation.
func (c *TestClient) StopInstance(project, zone, name string) error {
	if c.StopInstanceFn != nil {
		return c.StopInstanceFn(project, zone, name)
	}
	return c.client.StopInstance(project, zone, name)
}

// DeleteDisk uses the override method DeleteDiskFn or the real implementation.
func (c *TestClient) DeleteDisk(project, zone, name string) error {
	if c.DeleteDiskFn != nil {
		return c.DeleteDiskFn(project, zone, name)
	}
	return c.client.DeleteDisk(project, zone, name)
}

// DeleteForwardingRule uses the override method DeleteForwardingRuleFn or the real implementation.
func (c *TestClient) DeleteForwardingRule(project, region, name string) error {
	if c.DeleteForwardingRuleFn != nil {
		return c.DeleteForwardingRuleFn(project, region, name)
	}
	return c.client.DeleteForwardingRule(project, region, name)
}

// DeleteFirewallRule uses the override method DeleteFirewallRuleFn or the real implementation.
func (c *TestClient) DeleteFirewallRule(project, name string) error {
	if c.DeleteFirewallRuleFn != nil {
		return c.DeleteFirewallRuleFn(project, name)
	}
	return c.client.DeleteFirewallRule(project, name)
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

// DeleteNetwork uses the override method DeleteNetworkFn or the real implementation.
func (c *TestClient) DeleteNetwork(project, name string) error {
	if c.DeleteNetworkFn != nil {
		return c.DeleteNetworkFn(project, name)
	}
	return c.client.DeleteNetwork(project, name)
}

// DeleteSubnetwork uses the override method DeleteSubnetworkFn or the real implementation.
func (c *TestClient) DeleteSubnetwork(project, region, name string) error {
	if c.DeleteSubnetworkFn != nil {
		return c.DeleteSubnetworkFn(project, region, name)
	}
	return c.client.DeleteSubnetwork(project, region, name)
}

// DeleteTargetInstance uses the override method DeleteTargetInstanceFn or the real implementation.
func (c *TestClient) DeleteTargetInstance(project, zone, name string) error {
	if c.DeleteTargetInstanceFn != nil {
		return c.DeleteTargetInstanceFn(project, zone, name)
	}
	return c.client.DeleteTargetInstance(project, zone, name)
}

// DeprecateImage uses the override method DeprecateImageFn or the real implementation.
func (c *TestClient) DeprecateImage(project, name string, deprecationstatus *compute.DeprecationStatus) error {
	if c.DeprecateImageFn != nil {
		return c.DeprecateImageFn(project, name, deprecationstatus)
	}
	return c.client.DeprecateImage(project, name, deprecationstatus)
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
	if c.GetMachineTypeFn != nil {
		return c.GetMachineTypeFn(project, zone, machineType)
	}
	return c.client.GetMachineType(project, zone, machineType)
}

// ListMachineTypes uses the override method ListMachineTypesFn or the real implementation.
func (c *TestClient) ListMachineTypes(project, zone string, opts ...ListCallOption) ([]*compute.MachineType, error) {
	if c.ListMachineTypesFn != nil {
		return c.ListMachineTypesFn(project, zone, opts...)
	}
	return c.client.ListMachineTypes(project, zone, opts...)
}

// GetZone uses the override method GetZoneFn or the real implementation.
func (c *TestClient) GetZone(project, zone string) (*compute.Zone, error) {
	if c.GetZoneFn != nil {
		return c.GetZoneFn(project, zone)
	}
	return c.client.GetZone(project, zone)
}

// ListZones uses the override method ListZonesFn or the real implementation.
func (c *TestClient) ListZones(project string, opts ...ListCallOption) ([]*compute.Zone, error) {
	if c.ListZonesFn != nil {
		return c.ListZonesFn(project, opts...)
	}
	return c.client.ListZones(project, opts...)
}

// GetInstance uses the override method GetZoneFn or the real implementation.
func (c *TestClient) GetInstance(project, zone, name string) (*compute.Instance, error) {
	if c.GetInstanceFn != nil {
		return c.GetInstanceFn(project, zone, name)
	}
	return c.client.GetInstance(project, zone, name)
}

// ListInstances uses the override method ListInstancesFn or the real implementation.
func (c *TestClient) ListInstances(project, zone string, opts ...ListCallOption) ([]*compute.Instance, error) {
	if c.ListInstancesFn != nil {
		return c.ListInstancesFn(project, zone, opts...)
	}
	return c.client.ListInstances(project, zone, opts...)
}

// GetDisk uses the override method GetZoneFn or the real implementation.
func (c *TestClient) GetDisk(project, zone, name string) (*compute.Disk, error) {
	if c.GetDiskFn != nil {
		return c.GetDiskFn(project, zone, name)
	}
	return c.client.GetDisk(project, zone, name)
}

// ListDisks uses the override method ListDisksFn or the real implementation.
func (c *TestClient) ListDisks(project, zone string, opts ...ListCallOption) ([]*compute.Disk, error) {
	if c.ListDisksFn != nil {
		return c.ListDisksFn(project, zone, opts...)
	}
	return c.client.ListDisks(project, zone, opts...)
}

// GetForwardingRule uses the override method GetForwardingRuleFn or the real implementation.
func (c *TestClient) GetForwardingRule(project, region, name string) (*compute.ForwardingRule, error) {
	if c.GetForwardingRuleFn != nil {
		return c.GetForwardingRuleFn(project, region, name)
	}
	return c.client.GetForwardingRule(project, region, name)
}

// ListForwardingRules uses the override method ListForwardingRulesFn or the real implementation.
func (c *TestClient) ListForwardingRules(project, region string, opts ...ListCallOption) ([]*compute.ForwardingRule, error) {
	if c.ListForwardingRulesFn != nil {
		return c.ListForwardingRulesFn(project, region, opts...)
	}
	return c.client.ListForwardingRules(project, region, opts...)
}

// GetFirewallRule uses the override method GetFirewallRuleFn or the real implementation.
func (c *TestClient) GetFirewallRule(project, name string) (*compute.Firewall, error) {
	if c.GetFirewallRuleFn != nil {
		return c.GetFirewallRuleFn(project, name)
	}
	return c.client.GetFirewallRule(project, name)
}

// ListFirewallRules uses the override method ListFirewallRulesFn or the real implementation.
func (c *TestClient) ListFirewallRules(project string, opts ...ListCallOption) ([]*compute.Firewall, error) {
	if c.ListFirewallRulesFn != nil {
		return c.ListFirewallRulesFn(project, opts...)
	}
	return c.client.ListFirewallRules(project, opts...)
}

// GetImage uses the override method GetImageFn or the real implementation.
func (c *TestClient) GetImage(project, name string) (*computeAlpha.Image, error) {
	if c.GetImageFn != nil {
		return c.GetImageFn(project, name)
	}
	return c.client.GetImage(project, name)
}

// GetImageFromFamily uses the override method GetImageFromFamilyFn or the real implementation.
func (c *TestClient) GetImageFromFamily(project, family string) (*computeAlpha.Image, error) {
	if c.GetImageFromFamilyFn != nil {
		return c.GetImageFromFamilyFn(project, family)
	}
	return c.client.GetImageFromFamily(project, family)
}

// ListImages uses the override method ListImagesFn or the real implementation.
func (c *TestClient) ListImages(project string, opts ...ListCallOption) ([]*computeAlpha.Image, error) {
	if c.ListImagesFn != nil {
		return c.ListImagesFn(project, opts...)
	}
	return c.client.ListImages(project, opts...)
}

// GetLicense uses the override method GetLicenseFn or the real implementation.
func (c *TestClient) GetLicense(project, name string) (*compute.License, error) {
	if c.GetLicenseFn != nil {
		return c.GetLicenseFn(project, name)
	}
	return c.client.GetLicense(project, name)
}

// GetNetwork uses the override method GetNetworkFn or the real implementation.
func (c *TestClient) GetNetwork(project, name string) (*compute.Network, error) {
	if c.GetNetworkFn != nil {
		return c.GetNetworkFn(project, name)
	}
	return c.client.GetNetwork(project, name)
}

// ListNetworks uses the override method ListNetworksFn or the real implementation.
func (c *TestClient) ListNetworks(project string, opts ...ListCallOption) ([]*compute.Network, error) {
	if c.ListNetworksFn != nil {
		return c.ListNetworksFn(project, opts...)
	}
	return c.client.ListNetworks(project, opts...)
}

// GetSubnetwork uses the override method GetSubnetworkFn or the real implementation.
func (c *TestClient) GetSubnetwork(project, region, name string) (*compute.Subnetwork, error) {
	if c.GetSubnetworkFn != nil {
		return c.GetSubnetworkFn(project, region, name)
	}
	return c.client.GetSubnetwork(project, region, name)
}

// ListSubnetworks uses the override method ListSubnetworksFn or the real implementation.
func (c *TestClient) ListSubnetworks(project, region string, opts ...ListCallOption) ([]*compute.Subnetwork, error) {
	if c.ListSubnetworksFn != nil {
		return c.ListSubnetworksFn(project, region, opts...)
	}
	return c.client.ListSubnetworks(project, region, opts...)
}

// GetTargetInstance uses the override method GetTargetInstanceFn or the real implementation.
func (c *TestClient) GetTargetInstance(project, zone, name string) (*compute.TargetInstance, error) {
	if c.GetTargetInstanceFn != nil {
		return c.GetTargetInstanceFn(project, zone, name)
	}
	return c.client.GetTargetInstance(project, zone, name)
}

// ListTargetInstances uses the override method ListTargetInstancesFn or the real implementation.
func (c *TestClient) ListTargetInstances(project, zone string, opts ...ListCallOption) ([]*compute.TargetInstance, error) {
	if c.ListTargetInstancesFn != nil {
		return c.ListTargetInstancesFn(project, zone, opts...)
	}
	return c.client.ListTargetInstances(project, zone, opts...)
}

// GetSerialPortOutput uses the override method GetSerialPortOutputFn or the real implementation.
func (c *TestClient) GetSerialPortOutput(project, zone, name string, port, start int64) (*compute.SerialPortOutput, error) {
	if c.GetSerialPortOutputFn != nil {
		return c.GetSerialPortOutputFn(project, zone, name, port, start)
	}
	return c.client.GetSerialPortOutput(project, zone, name, port, start)
}

// GetGuestAttributes uses the override method GetGuestAttributesFn or the real implementation.
func (c *TestClient) GetGuestAttributes(project, zone, name, queryPath, variableKey string) (*computeBeta.GuestAttributes, error) {
	if c.GetGuestAttributesFn != nil {
		return c.GetGuestAttributesFn(project, zone, name, queryPath, variableKey)
	}
	return c.client.GetGuestAttributes(project, zone, name, queryPath, variableKey)
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

// ResizeDisk uses the override method ResizeDiskFn or the real implementation.
func (c *TestClient) ResizeDisk(project, zone, disk string, drr *compute.DisksResizeRequest) error {
	if c.ResizeDiskFn != nil {
		return c.ResizeDiskFn(project, zone, disk, drr)
	}
	return c.client.ResizeDisk(project, zone, disk, drr)
}

// SetInstanceMetadata uses the override method SetInstancemetadataFn or the real implementation.
func (c *TestClient) SetInstanceMetadata(project, zone, name string, md *compute.Metadata) error {
	if c.InstanceStoppedFn != nil {
		return c.SetInstanceMetadataFn(project, zone, name, md)
	}
	return c.client.SetInstanceMetadata(project, zone, name, md)
}

// SetCommonInstanceMetadata uses the override method SetCommonInstanceMetadataFn or the real implementation.
func (c *TestClient) SetCommonInstanceMetadata(project string, md *compute.Metadata) error {
	if c.InstanceStoppedFn != nil {
		return c.SetCommonInstanceMetadataFn(project, md)
	}
	return c.client.SetCommonInstanceMetadata(project, md)
}

// zoneOperationsWait uses the override method zoneOperationsWaitFn or the real implementation.
func (c *TestClient) zoneOperationsWait(project, zone, name string) error {
	if c.zoneOperationsWaitFn != nil {
		return c.zoneOperationsWaitFn(project, zone, name)
	}
	return c.client.zoneOperationsWait(project, zone, name)
}

// regionOperationsWait uses the override method regionOperationsWaitFn or the real implementation.
func (c *TestClient) regionOperationsWait(project, region, name string) error {
	if c.regionOperationsWaitFn != nil {
		return c.regionOperationsWaitFn(project, region, name)
	}
	return c.client.regionOperationsWait(project, region, name)
}

// globalOperationsWait uses the override method globalOperationsWaitFn or the real implementation.
func (c *TestClient) globalOperationsWait(project, name string) error {
	if c.globalOperationsWaitFn != nil {
		return c.globalOperationsWaitFn(project, name)
	}
	return c.client.globalOperationsWait(project, name)
}
