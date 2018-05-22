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
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
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
func instanceExists(client daisyCompute.Client, project, zone, instance string) (bool, dErr) {
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
			return false, errf("error listing instances for project %q: %v", project, err)
		}
		var instances []string
		for _, i := range il {
			instances = append(instances, i.Name)
		}
		instanceCache.exists[project][zone] = instances
	}
	return strIn(instance, instanceCache.exists[project][zone]), nil
}

// Instance is used to create a GCE instance. Output of serial port 1 will be streamed to the daisy logs directory.
type Instance struct {
	compute.Instance
	Resource

	// Additional metadata to set for the instance.
	Metadata map[string]string `json:"metadata,omitempty"`
	// OAuth2 scopes to give the instance. If none are specified
	// https://www.googleapis.com/auth/devstorage.read_only will be added.
	Scopes []string `json:",omitempty"`
	// StartupScript is the Sources path to a startup script to use in this step.
	// This will be automatically mapped to the appropriate metadata key.
	StartupScript string `json:",omitempty"`
}

// MarshalJSON is a hacky workaround to prevent Instance from using compute.Instance's implementation.
func (i *Instance) MarshalJSON() ([]byte, error) {
	return json.Marshal(*i)
}

func (i *Instance) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	i.Name, i.Zone, errs = i.Resource.populate(ctx, s, i.Name, i.Zone)
	i.Description = strOr(i.Description, fmt.Sprintf("Instance created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))

	errs = addErrs(errs, i.populateDisks(s.w))
	errs = addErrs(errs, i.populateMachineType())
	errs = addErrs(errs, i.populateMetadata(s.w))
	errs = addErrs(errs, i.populateNetworks())
	errs = addErrs(errs, i.populateScopes())
	i.link = fmt.Sprintf("projects/%s/zones/%s/instances/%s", i.Project, i.Zone, i.Name)
	return errs
}

func (i *Instance) populateDisks(w *Workflow) dErr {
	autonameIdx := 1
	for di, d := range i.Disks {
		d.Boot = di == 0 // TODO(crunkleton) should we do this?
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
		} else if d.DeviceName == "" {
			d.DeviceName = path.Base(d.Source)
		}
	}
	return nil
}

func (i *Instance) populateMachineType() dErr {
	i.MachineType = strOr(i.MachineType, "n1-standard-1")
	if machineTypeURLRegex.MatchString(i.MachineType) {
		i.MachineType = extendPartialURL(i.MachineType, i.Project)
	} else {
		i.MachineType = fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", i.Project, i.Zone, i.MachineType)
	}
	return nil
}

func (i *Instance) populateMetadata(w *Workflow) dErr {
	if i.Metadata == nil {
		i.Metadata = map[string]string{}
	}
	if i.Instance.Metadata == nil {
		i.Instance.Metadata = &compute.Metadata{}
	}
	i.Metadata["daisy-sources-path"] = "gs://" + path.Join(w.bucket, w.sourcesPath)
	i.Metadata["daisy-logs-path"] = "gs://" + path.Join(w.bucket, w.logsPath)
	i.Metadata["daisy-outs-path"] = "gs://" + path.Join(w.bucket, w.outsPath)
	if i.StartupScript != "" {
		if !w.sourceExists(i.StartupScript) {
			return errf("bad value for StartupScript, source not found: %s", i.StartupScript)
		}
		i.StartupScript = "gs://" + path.Join(w.bucket, w.sourcesPath, i.StartupScript)
		i.Metadata["startup-script-url"] = i.StartupScript
		i.Metadata["windows-startup-script-url"] = i.StartupScript
	}
	for k, v := range i.Metadata {
		vCopy := v
		i.Instance.Metadata.Items = append(i.Instance.Metadata.Items, &compute.MetadataItems{Key: k, Value: &vCopy})
	}
	return nil
}

func (i *Instance) populateNetworks() dErr {
	defaultAcs := []*compute.AccessConfig{{Type: defaultAccessConfigType}}

	if i.NetworkInterfaces == nil {
		i.NetworkInterfaces = []*compute.NetworkInterface{{}}
	}
	for _, n := range i.NetworkInterfaces {
		if n.AccessConfigs == nil {
			n.AccessConfigs = defaultAcs
		}
		n.Network = strOr(n.Network, "global/networks/default")
		if networkURLRegex.MatchString(n.Network) {
			n.Network = extendPartialURL(n.Network, i.Project)
		}
	}

	return nil
}

func (i *Instance) populateScopes() dErr {
	if len(i.Scopes) == 0 {
		i.Scopes = append(i.Scopes, "https://www.googleapis.com/auth/devstorage.read_only")
	}
	if i.ServiceAccounts == nil {
		i.ServiceAccounts = []*compute.ServiceAccount{{Email: "default", Scopes: i.Scopes}}
	}
	return nil
}

func (i *Instance) validate(ctx context.Context, s *Step) dErr {
	pre := fmt.Sprintf("cannot create instance %q", i.daisyName)
	errs := i.Resource.validateWithZone(ctx, s, i.Zone, pre)
	errs = addErrs(errs, i.validateDisks(s))
	errs = addErrs(errs, i.validateMachineType(s.w.ComputeClient))
	errs = addErrs(errs, i.validateNetworks(s))

	// Register creation.
	errs = addErrs(errs, s.w.instances.regCreate(i.daisyName, &i.Resource, s))
	return errs
}

func (i *Instance) validateDisks(s *Step) (errs dErr) {
	if len(i.Disks) == 0 {
		errs = addErrs(errs, errf("cannot create instance: no disks provided"))
	}

	for _, d := range i.Disks {
		if !checkDiskMode(d.Mode) {
			errs = addErrs(errs, errf("cannot create instance: bad disk mode: %q", d.Mode))
		}
		if d.Source != "" && d.InitializeParams != nil {
			errs = addErrs(errs, errf("cannot create instance: disk.source and disk.initializeParams are mutually exclusive"))
		}
		if d.InitializeParams != nil {
			errs = addErrs(errs, i.validateDiskInitializeParams(d, s))
		} else {
			errs = addErrs(errs, i.validateDiskSource(d, s))
		}
	}
	return
}

func (i *Instance) validateDiskInitializeParams(d *compute.AttachedDisk, s *Step) (errs dErr) {
	p := d.InitializeParams
	if !rfc1035Rgx.MatchString(p.DiskName) {
		errs = addErrs(errs, errf("cannot create instance: bad InitializeParams.DiskName: %q", p.DiskName))
	}
	if _, err := s.w.images.regUse(p.SourceImage, s); err != nil {
		errs = addErrs(errs, errf("cannot create instance: can't use InitializeParams.SourceImage %q: %v", p.SourceImage, err))
	}
	parts := namedSubexp(diskTypeURLRgx, p.DiskType)
	if parts["project"] != i.Project {
		errs = addErrs(errs, errf("cannot create instance in project %q with InitializeParams.DiskType in project %q", i.Project, parts["project"]))
	}
	if parts["zone"] != i.Zone {
		errs = addErrs(errs, errf("cannot create instance in zone %q with InitializeParams.DiskType in zone %q", i.Zone, parts["zone"]))
	}

	link := fmt.Sprintf("projects/%s/zones/%s/disks/%s", i.Project, i.Zone, p.DiskName)
	// Set cleanup if not being autodeleted.
	r := &Resource{RealName: p.DiskName, link: link, NoCleanup: d.AutoDelete}
	errs = addErrs(errs, s.w.disks.regCreate(p.DiskName, r, s, false))
	return
}

func (i *Instance) validateDiskSource(d *compute.AttachedDisk, s *Step) dErr {
	dr, errs := s.w.disks.regUse(d.Source, s)
	if dr == nil {
		// Return now, the rest of this function can't be run without dr.
		return addErrs(errs, errf("cannot create instance: disk %q not found in registry", d.Source))
	}

	// Ensure disk is in the same project and zone.
	result := namedSubexp(diskURLRgx, dr.link)
	if result["project"] != i.Project {
		errs = addErrs(errs, errf("cannot create instance in project %q with disk in project %q: %q", i.Project, result["project"], d.Source))
	}
	if result["zone"] != i.Zone {
		errs = addErrs(errs, errf("cannot create instance in project %q with disk in zone %q: %q", i.Zone, result["zone"], d.Source))
	}
	return errs
}

func (i *Instance) validateMachineType(client daisyCompute.Client) (errs dErr) {
	if !machineTypeURLRegex.MatchString(i.MachineType) {
		errs = addErrs(errs, errf("can't create instance: bad MachineType: %q", i.MachineType))
		return
	}

	result := namedSubexp(machineTypeURLRegex, i.MachineType)
	if result["project"] != i.Project {
		errs = addErrs(errs, errf("cannot create instance in project %q with MachineType in project %q: %q", i.Project, result["project"], i.MachineType))
	}
	if result["zone"] != i.Zone {
		errs = addErrs(errs, errf("cannot create instance in zone %q with MachineType in zone %q: %q", i.Zone, result["zone"], i.MachineType))
	}

	if exists, err := machineTypeExists(client, result["project"], result["zone"], result["machinetype"]); err != nil {
		errs = addErrs(errs, errf("cannot create instance, bad machineType lookup: %q, error: %v", result["machinetype"], err))
	} else if !exists {
		errs = addErrs(errs, errf("cannot create instance, machineType does not exist: %q", result["machinetype"]))
	}
	return
}

func (i *Instance) validateNetworks(s *Step) (errs dErr) {
	for _, n := range i.NetworkInterfaces {
		nr, err := s.w.networks.regUse(n.Network, s)
		if err != nil {
			errs = addErrs(errs, err)
			continue
		}

		// Ensure network is in the same project.
		result := namedSubexp(networkURLRegex, nr.link)
		if result["project"] != i.Project {
			errs = addErrs(errs, errf("cannot create instance in project %q with Network in project %q: %q", i.Project, result["project"], n.Network))
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
	ir.baseResourceRegistry.stopFn = ir.stopFn
	ir.init()
	return ir
}

func (ir *instanceRegistry) deleteFn(res *Resource) dErr {
	m := namedSubexp(instanceURLRgx, res.link)
	err := ir.w.ComputeClient.DeleteInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (ir *instanceRegistry) stopFn(res *Resource) dErr {
	m := namedSubexp(instanceURLRgx, res.link)
	err := ir.w.ComputeClient.StopInstance(m["project"], m["zone"], m["instance"])
	if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == http.StatusNotFound {
		return typedErr(resourceDNEError, err)
	}
	return newErr(err)
}

func (ir *instanceRegistry) regCreate(name string, res *Resource, s *Step) dErr {
	// Base creation logic.
	errs := ir.baseResourceRegistry.regCreate(name, res, s, false)

	// Find the Instance responsible for this.
	var i *Instance
	for _, i = range *s.CreateInstances {
		if &i.Resource == res {
			break
		}
	}
	// Register disk attachments.
	for _, d := range i.Disks {
		dName := d.Source
		if d.InitializeParams != nil {
			dName = d.InitializeParams.DiskName
		}
		errs = addErrs(errs, ir.w.disks.regAttach(dName, name, d.Mode, s))
	}

	// Register network connections.
	for _, n := range i.NetworkInterfaces {
		nName := n.Network
		errs = addErrs(errs, ir.w.networks.regConnect(nName, name, s))
	}
	return errs
}

func (ir *instanceRegistry) regDelete(name string, s *Step) dErr {
	errs := ir.baseResourceRegistry.regDelete(name, s)
	errs = addErrs(errs, ir.w.disks.regDetachAll(name, s))
	return addErrs(errs, ir.w.networks.regDisconnectAll(name, s))
}
