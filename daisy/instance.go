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
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	computeBeta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const (
	defaultAccessConfigType = "ONE_TO_ONE_NAT"
	defaultDiskMode         = diskModeRW
	defaultDiskType         = "pd-standard"
	diskModeRO              = "READ_ONLY"
	diskModeRW              = "READ_WRITE"
)

var (
	instanceCache struct {
		exists map[string]map[string][]string
		mu     sync.Mutex
	}
	instanceURLRgx = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/instances/(?P<instance>%[2]s)$`, projectRgxStr, rfc1035))
	validDiskModes = []string{diskModeRO, diskModeRW}
)

func checkDiskMode(m string) bool {
	parts := strings.Split(m, "/")
	m = parts[len(parts)-1]
	return strIn(m, validDiskModes)
}

// instanceExists should only be used during validation for existing GCE instances
// and should not be relied or populated for daisy created resources.
func instanceExists(client daisyCompute.Client, project, zone, instance string) (bool, DError) {
	instanceCache.mu.Lock()
	defer instanceCache.mu.Unlock()
	if instanceCache.exists == nil {
		instanceCache.exists = map[string]map[string][]string{}
	}
	if _, ok := instanceCache.exists[project]; !ok {
		instanceCache.exists[project] = map[string][]string{}
	}
	if _, ok := instanceCache.exists[project][zone]; !ok {
		il, err := client.ListInstances(project, zone)
		if err != nil {
			return false, Errf("error listing instances for project %q: %v", project, err)
		}
		var instances []string
		for _, i := range il {
			instances = append(instances, i.Name)
		}
		instanceCache.exists[project][zone] = instances
	}
	return strIn(instance, instanceCache.exists[project][zone]), nil
}

// MarshalJSON is a workaround to prevent Instance from using compute.Instance's implementation.
func (i *Instance) MarshalJSON() ([]byte, error) {
	return json.Marshal(*i)
}

// InstanceInterface represent abstract Instance across different API stages (Alpha, Beta, API)
type InstanceInterface interface {
	getBase() *InstanceBase
	getName() string
	setName(name string)
	getDescription() string
	setDescription(description string)
	getZone() string
	setZone(zone string)
	getMachineType() string
	setMachineType(machineType string)
	populateDisks(w *Workflow) DError
	populateNetworks() DError
	populateScopes() DError
	initializeComputeMetadata()
	appendComputeMetadata(key string, value *string)
	validateNetworks(s *Step) (errs DError)
	getComputeDisks() []*computeDisk
	create(cc daisyCompute.Client) error
	updateDisksAndNetworksBeforeCreate(w *Workflow)
	getMetadata() map[string]string
	setMetadata(md map[string]string)
	getSourceMachineImage() string
	setSourceMachineImage(machineImage string)
}

// InstanceBase is a base struct for GA/Beta instances.
// It holds the shared properties between the two.
type InstanceBase struct {
	Resource
	fallbackRetryableTask

	// OAuth2 scopes to give the instance. If left unset
	// https://www.googleapis.com/auth/devstorage.read_only will be added.
	Scopes []string `json:",omitempty"`
	// StartupScript is the Sources path to a startup script to use in this step.
	// This will be automatically mapped to the appropriate metadata key.
	StartupScript string `json:",omitempty"`
}

func (ib *InstanceBase) getFallbackRetryableTask() fallbackRetryableTask {
	return ib.fallbackRetryableTask
}

// Instance is used to create a GCE instance using GA API.
// Output of serial port 1 will be streamed to the daisy logs directory.
type Instance struct {
	InstanceBase
	compute.Instance

	// Additional metadata to set for the instance.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// InstanceBeta is used to create a GCE instance using Beta API.
// Output of serial port 1 will be streamed to the daisy logs directory.
type InstanceBeta struct {
	InstanceBase
	computeBeta.Instance

	// Additional metadata to set for the instance.
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (i *Instance) getBase() *InstanceBase {
	return &i.InstanceBase
}

func (i *Instance) getMachineType() string {
	return i.MachineType
}

func (i *Instance) setMachineType(machineType string) {
	i.MachineType = machineType
}

func (i *Instance) getDescription() string {
	return i.Description
}
func (i *Instance) setDescription(description string) {
	i.Description = description
}

func (i *Instance) getName() string {
	return i.Name
}
func (i *Instance) setName(name string) {
	i.Name = name
}

func (i *Instance) getZone() string {
	return i.Zone
}

func (i *Instance) setZone(zone string) {
	i.Zone = zone
}

func (i *Instance) initializeComputeMetadata() {
	if i.Instance.Metadata == nil {
		i.Instance.Metadata = &compute.Metadata{}
	}
}

func (i *Instance) appendComputeMetadata(key string, value *string) {
	i.Instance.Metadata.Items = append(i.Instance.Metadata.Items, &compute.MetadataItems{Key: key, Value: value})
}

func (i *Instance) create(cc daisyCompute.Client) error {
	return cc.CreateInstance(i.Project, i.Zone, &i.Instance)
}

func (i *Instance) updateDisksAndNetworksBeforeCreate(w *Workflow) {
	for _, d := range i.Disks {
		if diskRes, ok := w.disks.get(d.Source); ok {
			d.Source = diskRes.link
		}
		if d.InitializeParams != nil && d.InitializeParams.SourceImage != "" {
			if image, ok := w.images.get(d.InitializeParams.SourceImage); ok {
				d.InitializeParams.SourceImage = image.link
			}
		}
	}

	for _, n := range i.NetworkInterfaces {
		if netRes, ok := w.networks.get(n.Network); ok {
			n.Network = netRes.link
		}
		if subnetRes, ok := w.subnetworks.get(n.Subnetwork); ok {
			n.Subnetwork = subnetRes.link
		}
	}
}

func (i *Instance) getMetadata() map[string]string {
	return i.Metadata
}

func (i *Instance) setMetadata(md map[string]string) {
	i.Metadata = md
}

func (i *Instance) getSourceMachineImage() string {
	return ""
}

func (i *Instance) setSourceMachineImage(machineImage string) {}

func (i *Instance) register(name string, s *Step, ir *instanceRegistry, errs DError) {
	// Register disk attachments.
	for _, d := range i.Disks {
		dName := d.Source
		if d.InitializeParams != nil {
			parts := namedSubexp(diskTypeURLRgx, d.InitializeParams.DiskType)
			if parts["disktype"] == "local-ssd" {
				continue
			}
			dName = d.InitializeParams.DiskName
		}
		errs = addErrs(errs, ir.w.disks.regAttach(dName, name, d.Mode, s))
	}

	// Register network connections.
	for _, n := range i.NetworkInterfaces {
		nName := n.Network
		errs = addErrs(errs, ir.w.networks.regConnect(nName, name, s))
	}
}

func (i *InstanceBeta) getBase() *InstanceBase {
	return &i.InstanceBase
}

func (i *InstanceBeta) getMachineType() string {
	return i.MachineType
}

func (i *InstanceBeta) setMachineType(machineType string) {
	i.MachineType = machineType
}

func (i *InstanceBeta) getDescription() string {
	return i.Description
}
func (i *InstanceBeta) setDescription(description string) {
	i.Description = description
}

func (i *InstanceBeta) getName() string {
	return i.Name
}
func (i *InstanceBeta) setName(name string) {
	i.Name = name
}

func (i *InstanceBeta) getZone() string {
	return i.Zone
}

func (i *InstanceBeta) setZone(zone string) {
	i.Zone = zone
}

func (i *InstanceBeta) appendComputeMetadata(key string, value *string) {
	i.Instance.Metadata.Items = append(i.Instance.Metadata.Items, &computeBeta.MetadataItems{Key: key, Value: value})
}

func (i *InstanceBeta) initializeComputeMetadata() {
	if i.Instance.Metadata == nil {
		i.Instance.Metadata = &computeBeta.Metadata{}
	}
}

func (i *InstanceBeta) create(cc daisyCompute.Client) error {
	return cc.CreateInstanceBeta(i.Project, i.Zone, &i.Instance)
}

func (i *InstanceBeta) updateDisksAndNetworksBeforeCreate(w *Workflow) {
	for _, d := range i.Disks {
		if diskRes, ok := w.disks.get(d.Source); ok {
			d.Source = diskRes.link
		}
		if d.InitializeParams != nil && d.InitializeParams.SourceImage != "" {
			if image, ok := w.images.get(d.InitializeParams.SourceImage); ok {
				d.InitializeParams.SourceImage = image.link
			}
		}
	}

	for _, n := range i.NetworkInterfaces {
		if netRes, ok := w.networks.get(n.Network); ok {
			n.Network = netRes.link
		}
		if subnetRes, ok := w.subnetworks.get(n.Subnetwork); ok {
			n.Subnetwork = subnetRes.link
		}
	}
}

func (i *InstanceBeta) getMetadata() map[string]string {
	return i.Metadata
}

func (i *InstanceBeta) setMetadata(md map[string]string) {
	i.Metadata = md
}

func (i *InstanceBeta) getSourceMachineImage() string {
	return i.Instance.SourceMachineImage
}

func (i *InstanceBeta) setSourceMachineImage(machineImage string) {
	i.SourceMachineImage = machineImage
}

func (i *InstanceBeta) register(name string, s *Step, ir *instanceRegistry, errs DError) {
	// Register disk attachments.
	for _, d := range i.Disks {
		dName := d.Source
		if d.InitializeParams != nil {
			parts := namedSubexp(diskTypeURLRgx, d.InitializeParams.DiskType)
			if parts["disktype"] == "local-ssd" {
				continue
			}
			dName = d.InitializeParams.DiskName
		}
		errs = addErrs(errs, ir.w.disks.regAttach(dName, name, d.Mode, s))
	}

	// Register network connections.
	for _, n := range i.NetworkInterfaces {
		nName := n.Network
		errs = addErrs(errs, ir.w.networks.regConnect(nName, name, s))
	}
}

func (ib *InstanceBase) populate(ctx context.Context, ii InstanceInterface, s *Step) DError {
	name, zone, errs := ib.Resource.populateWithZone(ctx, s, ii.getName(), ii.getZone())
	ii.setName(name)
	ii.setZone(zone)

	ii.setDescription(strOr(ii.getDescription(), fmt.Sprintf("Instance created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username)))
	errs = addErrs(errs, ii.populateDisks(s.w))
	errs = addErrs(errs, ib.populateMachineType(ii))
	errs = addErrs(errs, ib.populateMetadata(ii, s.w))
	errs = addErrs(errs, ii.populateNetworks())
	errs = addErrs(errs, ii.populateScopes())
	ib.link = fmt.Sprintf("projects/%s/zones/%s/instances/%s", ib.Project, ii.getZone(), ii.getName())

	if machineImageURLRgx.MatchString(ii.getSourceMachineImage()) {
		ii.setSourceMachineImage(extendPartialURL(ii.getSourceMachineImage(), ib.Project))
	}
	return errs
}

func (i *Instance) populateDisks(w *Workflow) DError {
	autonameIdx := 1
	for di, d := range i.Disks {
		d.Boot = di == 0
		d.Mode = strOr(d.Mode, defaultDiskMode)
		if diskURLRgx.MatchString(d.Source) {
			d.Source = extendPartialURL(d.Source, i.Project)
		}
		p := d.InitializeParams
		if p != nil {
			// If name isn't set, set name to "instance-name", "instance-name-2", etc.
			if p.DiskName == "" {
				p.DiskName = i.Name
				if autonameIdx > 1 {
					p.DiskName = fmt.Sprintf("%s-%d", i.Name, autonameIdx)
				}
				autonameIdx++
			}
			if d.DeviceName == "" {
				d.DeviceName = p.DiskName
			}

			// Extend SourceImage if short URL.
			if imageURLRgx.MatchString(p.SourceImage) {
				p.SourceImage = extendPartialURL(p.SourceImage, i.Project)
			}

			// Extend DiskType if short URL, or create extended URL.
			p.DiskType = strOr(p.DiskType, defaultDiskType)
			if diskTypeURLRgx.MatchString(p.DiskType) {
				p.DiskType = extendPartialURL(p.DiskType, i.Project)
			} else {
				p.DiskType = fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", i.Project, i.Zone, p.DiskType)
			}
			parts := namedSubexp(diskTypeURLRgx, p.DiskType)
			if parts["disktype"] == "local-ssd" {
				d.AutoDelete = true
				d.Type = "SCRATCH"
				p.DiskName = ""
			}
		} else if d.DeviceName == "" {
			d.DeviceName = path.Base(d.Source)
		}
	}
	return nil
}

func (i *InstanceBeta) populateDisks(w *Workflow) DError {
	autonameIdx := 1
	for di, d := range i.Disks {
		d.Boot = di == 0
		d.Mode = strOr(d.Mode, defaultDiskMode)
		if diskURLRgx.MatchString(d.Source) {
			d.Source = extendPartialURL(d.Source, i.Project)
		}
		p := d.InitializeParams
		if p != nil {
			// If name isn't set, set name to "instance-name", "instance-name-2", etc.
			if p.DiskName == "" {
				p.DiskName = i.Name
				if autonameIdx > 1 {
					p.DiskName = fmt.Sprintf("%s-%d", i.Name, autonameIdx)
				}
				autonameIdx++
			}
			if d.DeviceName == "" {
				d.DeviceName = p.DiskName
			}

			// Extend SourceImage if short URL.
			if imageURLRgx.MatchString(p.SourceImage) {
				p.SourceImage = extendPartialURL(p.SourceImage, i.Project)
			}

			// Extend DiskType if short URL, or create extended URL.
			p.DiskType = strOr(p.DiskType, defaultDiskType)
			if diskTypeURLRgx.MatchString(p.DiskType) {
				p.DiskType = extendPartialURL(p.DiskType, i.Project)
			} else {
				p.DiskType = fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", i.Project, i.Zone, p.DiskType)
			}
			parts := namedSubexp(diskTypeURLRgx, p.DiskType)
			if parts["disktype"] == "local-ssd" {
				d.AutoDelete = true
				d.Type = "SCRATCH"
				p.DiskName = ""
			}
		} else if d.DeviceName == "" {
			d.DeviceName = path.Base(d.Source)
		}
	}
	return nil
}

func (ib *InstanceBase) populateMachineType(ii InstanceInterface) DError {
	ii.setMachineType(strOr(ii.getMachineType(), "n1-standard-1"))
	if machineTypeURLRegex.MatchString(ii.getMachineType()) {
		ii.setMachineType(extendPartialURL(ii.getMachineType(), ib.Project))
	} else {
		ii.setMachineType(fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", ib.Project, ii.getZone(), ii.getMachineType()))
	}
	return nil
}

func (ib *InstanceBase) populateMetadata(ii InstanceInterface, w *Workflow) DError {
	if ii.getMetadata() == nil {
		ii.setMetadata(map[string]string{})
	}
	ii.initializeComputeMetadata()

	ii.getMetadata()["daisy-sources-path"] = "gs://" + path.Join(w.bucket, w.sourcesPath)
	ii.getMetadata()["daisy-logs-path"] = "gs://" + path.Join(w.bucket, w.logsPath)
	ii.getMetadata()["daisy-outs-path"] = "gs://" + path.Join(w.bucket, w.outsPath)
	if ib.StartupScript != "" {
		if !w.sourceExists(ib.StartupScript) {
			return Errf("bad value for StartupScript, source not found: %s", ib.StartupScript)
		}
		ib.StartupScript = "gs://" + path.Join(w.bucket, w.sourcesPath, ib.StartupScript)
		ii.getMetadata()["startup-script-url"] = ib.StartupScript
		ii.getMetadata()["windows-startup-script-url"] = ib.StartupScript
	}
	for k, v := range ii.getMetadata() {
		vCopy := v
		ii.appendComputeMetadata(k, &vCopy)
	}
	return nil
}

func (i *Instance) populateNetworks() DError {
	defaultAcs := []*compute.AccessConfig{{Type: defaultAccessConfigType}}

	if i.NetworkInterfaces == nil {
		i.NetworkInterfaces = []*compute.NetworkInterface{{}}
	}
	for _, n := range i.NetworkInterfaces {
		if n.AccessConfigs == nil {
			n.AccessConfigs = defaultAcs
		}

		// Only set deafult if no subnetwork or network set.
		if n.Subnetwork == "" {
			n.Network = strOr(n.Network, "global/networks/default")
		}

		if networkURLRegex.MatchString(n.Network) {
			n.Network = extendPartialURL(n.Network, i.Project)
		}

		if subnetworkURLRegex.MatchString(n.Subnetwork) {
			n.Subnetwork = extendPartialURL(n.Subnetwork, i.Project)
		}
	}

	return nil
}

func (i *InstanceBeta) populateNetworks() DError {
	defaultAcs := []*computeBeta.AccessConfig{{Type: defaultAccessConfigType}}

	if i.NetworkInterfaces == nil {
		i.NetworkInterfaces = []*computeBeta.NetworkInterface{{}}
	}
	for _, n := range i.NetworkInterfaces {
		if n.AccessConfigs == nil {
			n.AccessConfigs = defaultAcs
		}

		// Only set deafult if no subnetwork or network set.
		if n.Subnetwork == "" {
			n.Network = strOr(n.Network, "global/networks/default")
		}

		if networkURLRegex.MatchString(n.Network) {
			n.Network = extendPartialURL(n.Network, i.Project)
		}

		if subnetworkURLRegex.MatchString(n.Subnetwork) {
			n.Subnetwork = extendPartialURL(n.Subnetwork, i.Project)
		}
	}

	return nil
}

func (i *Instance) populateScopes() DError {
	if i.Scopes == nil {
		i.Scopes = append(i.Scopes, "https://www.googleapis.com/auth/devstorage.read_only")
	}
	if i.ServiceAccounts == nil {
		i.ServiceAccounts = []*compute.ServiceAccount{{Email: "default", Scopes: i.Scopes}}
	}
	return nil
}

func (i *InstanceBeta) populateScopes() DError {
	if i.Scopes == nil {
		i.Scopes = append(i.Scopes, "https://www.googleapis.com/auth/devstorage.read_only")
	}
	if i.ServiceAccounts == nil {
		i.ServiceAccounts = []*computeBeta.ServiceAccount{{Email: "default", Scopes: i.Scopes}}
	}
	return nil
}

func (ib *InstanceBase) validate(ctx context.Context, ii InstanceInterface, s *Step) DError {
	pre := fmt.Sprintf("cannot create instance %q", ib.daisyName)
	errs := ib.Resource.validateWithZone(ctx, s, ii.getZone(), pre)
	errs = addErrs(errs, ib.validateDisks(ii, s))
	errs = addErrs(errs, ib.validateMachineType(ii, s.w.ComputeClient))
	errs = addErrs(errs, ii.validateNetworks(s))
	errs = addErrs(errs, ib.validateSourceMachineImage(ii, s))

	// Register creation.
	errs = addErrs(errs, s.w.instances.regCreate(ib.daisyName, &ib.Resource, s))
	return errs
}

func (ib *InstanceBase) validateSourceMachineImage(ii InstanceInterface, s *Step) DError {
	// regUse needs the partal url of a non daisy resource.
	lookup := ii.getSourceMachineImage()
	if lookup == "" {
		return nil
	}
	if _, err := s.w.machineImages.regUse(lookup, s); err != nil {
		return newErr("failed to register use of machine image when creating an instance", err)
	}
	return nil
}

type computeDisk struct {
	mode                string
	source              string
	hasInitializeParams bool
	diskName            string
	sourceImage         string
	autoDelete          bool
	diskType            string
}

func (i *Instance) getComputeDisks() []*computeDisk {
	var computeDisks []*computeDisk
	for _, d := range i.Disks {
		computeDisk := computeDisk{mode: d.Mode, source: d.Source, hasInitializeParams: d.InitializeParams != nil, autoDelete: d.AutoDelete}
		if computeDisk.hasInitializeParams {
			computeDisk.diskName = d.InitializeParams.DiskName
			computeDisk.sourceImage = d.InitializeParams.SourceImage
			computeDisk.diskType = d.InitializeParams.DiskType
		}
		computeDisks = append(computeDisks, &computeDisk)
	}
	return computeDisks
}

func (i *InstanceBeta) getComputeDisks() []*computeDisk {
	var computeDisks []*computeDisk
	for _, d := range i.Disks {
		computeDisk := computeDisk{mode: d.Mode, source: d.Source, hasInitializeParams: d.InitializeParams != nil, autoDelete: d.AutoDelete}
		if computeDisk.hasInitializeParams {
			computeDisk.diskName = d.InitializeParams.DiskName
			computeDisk.sourceImage = d.InitializeParams.SourceImage
			computeDisk.diskType = d.InitializeParams.DiskType
		}
		computeDisks = append(computeDisks, &computeDisk)
	}
	return computeDisks
}

func (ib *InstanceBase) validateDisks(ii InstanceInterface, s *Step) (errs DError) {
	computeDisks := ii.getComputeDisks()
	if len(computeDisks) == 0 && ii.getSourceMachineImage() == "" {
		errs = addErrs(errs, Errf("cannot create instance: no disks nor source machine image provided"))
	}
	if len(computeDisks) > 0 && ii.getSourceMachineImage() != "" {
		errs = addErrs(errs, Errf("cannot create instance: can't provide disks when SourceMachineImage provided"))
	}
	for _, d := range computeDisks {
		if !checkDiskMode(d.mode) {
			errs = addErrs(errs, Errf("cannot create instance: bad disk mode: %q", d.mode))
		}
		if d.source != "" && d.hasInitializeParams {
			errs = addErrs(errs, Errf("cannot create instance: disk.source and disk.initializeParams are mutually exclusive"))
		}
		if d.hasInitializeParams {
			errs = addErrs(errs, ib.validateDiskInitializeParams(d, ii, s))
		} else {
			errs = addErrs(errs, ib.validateDiskSource(d.source, ii, s))
		}
	}
	return
}

func (ib *InstanceBase) validateDiskInitializeParams(d *computeDisk, ii InstanceInterface, s *Step) (errs DError) {
	parts := namedSubexp(diskTypeURLRgx, d.diskType)
	if parts["project"] != ib.Project {
		errs = addErrs(errs, Errf("cannot create instance in project %q with InitializeParams.DiskType in project %q", ib.Project, parts["project"]))
	}
	if parts["zone"] != ii.getZone() {
		errs = addErrs(errs, Errf("cannot create instance in zone %q with InitializeParams.DiskType in zone %q", ii.getZone(), parts["zone"]))
	}
	if parts["disktype"] == "local-ssd" {
		return
	}

	if _, err := s.w.images.regUse(d.sourceImage, s); err != nil {
		errs = addErrs(errs, Errf("cannot create instance: can't use InitializeParams.SourceImage %q: %v", d.sourceImage, err))
	}
	if !rfc1035Rgx.MatchString(d.diskName) {
		errs = addErrs(errs, Errf("cannot create instance: bad InitializeParams.DiskName: %q", d.diskName))
	}
	link := fmt.Sprintf("projects/%s/zones/%s/disks/%s", ib.Project, ii.getZone(), d.diskName)
	// Set cleanup if not being autodeleted.
	r := &Resource{RealName: d.diskName, link: link, NoCleanup: d.autoDelete}
	errs = addErrs(errs, s.w.disks.regCreate(d.diskName, r, s, false))

	return
}

func (ib *InstanceBase) validateDiskSource(diskSource string, ii InstanceInterface, s *Step) DError {
	dr, errs := s.w.disks.regUse(diskSource, s)
	if dr == nil {
		// Return now, the rest of this function can't be run without dr.
		return addErrs(errs, Errf("cannot create instance: disk %q not found in registry", diskSource))
	}

	// Ensure disk is in the same project and zone.
	result := namedSubexp(diskURLRgx, dr.link)
	if result["project"] != ib.Project {
		errs = addErrs(errs, Errf("cannot create instance in project %q with disk in project %q: %q", ib.Project, result["project"], diskSource))
	}
	if result["zone"] != ii.getZone() {
		errs = addErrs(errs, Errf("cannot create instance in project %q with disk in zone %q: %q", ii.getZone(), result["zone"], diskSource))
	}
	return errs
}

func (ib *InstanceBase) validateMachineType(ii InstanceInterface, client daisyCompute.Client) (errs DError) {
	if !machineTypeURLRegex.MatchString(ii.getMachineType()) {
		errs = addErrs(errs, Errf("can't create instance: bad MachineType: %q", ii.getMachineType()))
		return
	}

	result := namedSubexp(machineTypeURLRegex, ii.getMachineType())
	if result["project"] != ib.Project {
		errs = addErrs(errs, Errf("cannot create instance in project %q with MachineType in project %q: %q", ib.Project, result["project"], ii.getMachineType()))
	}
	if result["zone"] != ii.getZone() {
		errs = addErrs(errs, Errf("cannot create instance in zone %q with MachineType in zone %q: %q", ii.getZone(), result["zone"], ii.getMachineType()))
	}

	if exists, err := machineTypeExists(client, result["project"], result["zone"], result["machinetype"]); err != nil {
		errs = addErrs(errs, Errf("cannot create instance, bad machineType lookup: %q, error: %v", result["machinetype"], err))
	} else if !exists {
		errs = addErrs(errs, Errf("cannot create instance, machineType does not exist: %q", result["machinetype"]))
	}
	return
}

func (i *Instance) validateNetworks(s *Step) (errs DError) {
	for _, n := range i.NetworkInterfaces {
		if n.Subnetwork != "" {
			_, err := s.w.subnetworks.regUse(n.Subnetwork, s)
			if err != nil {
				errs = addErrs(errs, err)
			}
		}

		if n.Network != "" {
			_, err := s.w.networks.regUse(n.Network, s)
			if err != nil {
				errs = addErrs(errs, err)
				continue
			}
		}
	}
	return
}

func (i *InstanceBeta) validateNetworks(s *Step) (errs DError) {
	for _, n := range i.NetworkInterfaces {
		if n.Subnetwork != "" {
			_, err := s.w.subnetworks.regUse(n.Subnetwork, s)
			if err != nil {
				errs = addErrs(errs, err)
			}
		}

		if n.Network != "" {
			_, err := s.w.networks.regUse(n.Network, s)
			if err != nil {
				errs = addErrs(errs, err)
				continue
			}
		}
	}
	return
}

type instanceRegistry struct {
	baseResourceRegistry
}

func newInstanceRegistry(w *Workflow) *instanceRegistry {
	ir := &instanceRegistry{baseResourceRegistry: baseResourceRegistry{w: w, typeName: "instance", urlRgx: instanceURLRgx}}
	ir.baseResourceRegistry.deleteFn = ir.deleteFn
	ir.baseResourceRegistry.startFn = ir.startFn
	ir.baseResourceRegistry.stopFn = ir.stopFn
	ir.init()
	return ir
}

// SleepFn function is mocked on testing.
var SleepFn = time.Sleep

func (ir *instanceRegistry) deleteFn(res *Resource) DError {
	m := namedSubexp(instanceURLRgx, res.link)
	for i := 1; i < 4; i++ {
		if _, err := ir.w.ComputeClient.GetInstance(m["project"], m["zone"], m["instance"]); err != nil {
			// Can't remove an instance that was not even yet created!
			// However as the command was already submitted, wait.
			SleepFn((time.Duration(rand.Intn(1000))*time.Millisecond + 1*time.Second) * time.Duration(i))
			continue
		}
	}
	// Proceed to instance deletion
	err := ir.w.ComputeClient.DeleteInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to delete instance", err)
	}
	return newErr("failed to delete instance", err)
}

func (ir *instanceRegistry) startFn(res *Resource) DError {
	m := namedSubexp(instanceURLRgx, res.link)
	err := ir.w.ComputeClient.StartInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to start instance", err)
	}
	return newErr("failed to start instance", err)
}

func (ir *instanceRegistry) stopFn(res *Resource) DError {
	m := namedSubexp(instanceURLRgx, res.link)
	err := ir.w.ComputeClient.StopInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, "failed to stop instance", err)
	}
	return newErr("failed to stop instance", err)
}

func (ir *instanceRegistry) regCreate(name string, res *Resource, s *Step) DError {
	// Base creation logic.
	errs := ir.baseResourceRegistry.regCreate(name, res, s, false)

	// Find the Instance responsible for this.
	for _, i := range (*s.CreateInstances).Instances {
		if &i.Resource == res {
			i.register(name, s, ir, errs)
			return errs
		}
	}
	for _, i := range (*s.CreateInstances).InstancesBeta {
		if &i.Resource == res {
			i.register(name, s, ir, errs)
			return errs
		}
	}

	return errs
}

func (ir *instanceRegistry) regDelete(name string, s *Step) DError {
	errs := ir.baseResourceRegistry.regDelete(name, s)
	errs = addErrs(errs, ir.w.disks.regDetachAll(name, s))
	return addErrs(errs, ir.w.networks.regDisconnectAll(name, s))
}
