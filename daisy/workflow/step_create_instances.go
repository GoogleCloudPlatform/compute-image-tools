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

package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sync"
	"time"

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances []*CreateInstance

// CreateInstance creates a GCE instance. Output of serial port 1 will be
// streamed to the daisy logs directory.
type CreateInstance struct {
	compute.Instance

	// Additional metadata to set for the instance.
	Metadata map[string]string `json:"metadata,omitempty"`
	// OAuth2 scopes to give the instance. If none are specified
	// https://www.googleapis.com/auth/devstorage.read_only will be added.
	Scopes []string `json:",omitempty"`

	// StartupScript is the Sources path to a startup script to use in this step.
	// This will be automatically mapped to the appropriate metadata key.
	StartupScript string `json:",omitempty"`
	// Project to create the instance in, overrides workflow Project.
	Project string `json:",omitempty"`
	// Zone to create the instance in, overrides workflow Zone.
	Zone string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual resource name?
	ExactName bool

	// The name of the disk as known internally to Daisy.
	daisyName string
}

// MarshalJSON is a hacky workaround to prevent CreateInstance from using
// compute.Instance's implementation.
func (c *CreateInstance) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

func logSerialOutput(ctx context.Context, w *Workflow, name string, port int64, interval time.Duration) {
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.logger.Printf("CreateInstances: streaming instance %q serial port %d output to gs://%s/%s", name, port, w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	var errs int
	tick := time.Tick(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				// Instance was deleted by this workflow.
				if _, ok := instances[w].get(name); !ok {
					return
				}
				// Instance is stopped.
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if stopped && sErr == nil {
					return
				}
				// Otherwise retry 3 times on 5xx error.
				if apiErr, ok := err.(*googleapi.Error); ok && errs < 3 && (apiErr.Code >= 500 && apiErr.Code <= 599) {
					errs++
					continue
				}
				w.logger.Printf("CreateInstances: instance %q: error getting serial port: %v", name, err)
				return
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(ctx)
			wc.ContentType = "text/plain"
			if _, err := wc.Write(buf.Bytes()); err != nil {
				w.logger.Printf("CreateInstances: instance %q: error writing log to GCS: %v", name, err)
				return
			}
			if err := wc.Close(); err != nil {
				if apiErr, ok := err.(*googleapi.Error); ok && (apiErr.Code >= 500 && apiErr.Code <= 599) {
					errs++
					continue
				}
				w.logger.Printf("CreateInstances: instance %q: error saving log to GCS: %v", name, err)
				return
			}
			errs = 0
		}
	}
}

func (c *CreateInstance) populateDisks(w *Workflow) error {
	for i, d := range c.Disks {
		d.Boot = i == 0 // TODO(crunkleton) should we do this?
		d.Mode = strOr(d.Mode, "READ_WRITE")
		if diskURLRgx.MatchString(d.Source) {
			d.Source = extendPartialURL(d.Source, c.Project)
		}
	}
	return nil
}

func (c *CreateInstance) populateMachineType() error {
	c.MachineType = strOr(c.MachineType, "n1-standard-1")
	if machineTypeURLRegex.MatchString(c.MachineType) {
		c.MachineType = extendPartialURL(c.MachineType, c.Project)
	} else {
		c.MachineType = fmt.Sprintf("projects/%s/zones/%s/machineTypes/%s", c.Project, c.Zone, c.MachineType)
	}
	return nil
}

func (c *CreateInstance) populateMetadata(w *Workflow) error {
	if c.Metadata == nil {
		c.Metadata = map[string]string{}
	}
	if c.Instance.Metadata == nil {
		c.Instance.Metadata = &compute.Metadata{}
	}
	c.Metadata["daisy-sources-path"] = "gs://" + path.Join(w.bucket, w.sourcesPath)
	c.Metadata["daisy-logs-path"] = "gs://" + path.Join(w.bucket, w.logsPath)
	c.Metadata["daisy-outs-path"] = "gs://" + path.Join(w.bucket, w.outsPath)
	if c.StartupScript != "" {
		if !w.sourceExists(c.StartupScript) {
			return fmt.Errorf("bad value for StartupScript, source not found: %s", c.StartupScript)
		}
		c.StartupScript = "gs://" + path.Join(w.bucket, w.sourcesPath, c.StartupScript)
		c.Metadata["startup-script-url"] = c.StartupScript
		c.Metadata["windows-startup-script-url"] = c.StartupScript
	}
	for k, v := range c.Metadata {
		vCopy := v
		c.Instance.Metadata.Items = append(c.Instance.Metadata.Items, &compute.MetadataItems{Key: k, Value: &vCopy})
	}
	return nil
}

func (c *CreateInstance) populateNetworks() error {
	defaultAcs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	if c.NetworkInterfaces == nil {
		c.NetworkInterfaces = []*compute.NetworkInterface{{AccessConfigs: defaultAcs, Network: "global/networks/default"}}
	} else {
		for _, n := range c.NetworkInterfaces {
			if n.AccessConfigs == nil {
				n.AccessConfigs = defaultAcs
			}
			if !networkURLRegex.MatchString(n.Network) {
				n.Network = path.Join("global/networks", n.Network)
			}
		}
	}
	return nil
}

func (c *CreateInstance) populateScopes() error {
	if len(c.Scopes) == 0 {
		c.Scopes = append(c.Scopes, "https://www.googleapis.com/auth/devstorage.read_only")
	}
	if c.ServiceAccounts == nil {
		c.ServiceAccounts = []*compute.ServiceAccount{{Email: "default", Scopes: c.Scopes}}
	}
	return nil
}

// populate preprocesses fields: Name, Project, Zone, Description, MachineType, NetworkInterfaces, Scopes, ServiceAccounts, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateInstances) populate(ctx context.Context, s *Step) error {
	errs := []error{}
	for _, ci := range *c {
		// General fields preprocessing.
		ci.daisyName = ci.Name
		if !ci.ExactName {
			ci.Name = s.w.genName(ci.Name)
		}
		ci.Project = strOr(ci.Project, s.w.Project)
		ci.Zone = strOr(ci.Zone, s.w.Zone)
		ci.Description = strOr(ci.Description, fmt.Sprintf("Instance created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))

		errs = append(errs, ci.populateDisks(s.w))
		errs = append(errs, ci.populateMachineType())
		errs = append(errs, ci.populateMetadata(s.w))
		errs = append(errs, ci.populateNetworks())
		errs = append(errs, ci.populateScopes())
	}

	errStrs := []string{}
	for _, e := range errs {
		if e != nil {
			errStrs = append(errStrs, e.Error())
		}
	}
	if len(errStrs) > 0 {
		msg := fmt.Sprintf("%d errors:", len(errStrs))
		for _, s := range errStrs {
			msg += fmt.Sprintf("\n* %s", s)
		}
		return errors.New(msg)
	}
	return nil
}

func (c *CreateInstance) validateDisks(ctx context.Context, s *Step) (errs []error) {
	if len(c.Disks) == 0 {
		errs = append(errs, errors.New("cannot create instance: no disks provided"))
	}

	for i, d := range c.Disks {
		d.Boot = i == 0
		dr, err := disks[s.w].registerUsage(d.Source, s)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		d.Mode = strOr(d.Mode, "READ_WRITE")
		if !strIn(d.Mode, []string{"READ_ONLY", "READ_WRITE"}) {
			errs = append(errs, fmt.Errorf("cannot create instance: bad disk mode: %q", d.Mode))
		}

		// Ensure disk is in the same project and zone.
		match := diskURLRgx.FindStringSubmatch(dr.link)
		result := map[string]string{}
		for i, name := range diskURLRgx.SubexpNames() {
			if i != 0 {
				result[name] = match[i]
			}
		}

		if result["project"] != "" && result["project"] != c.Project {
			errs = append(errs, fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", c.Project, result["project"], d.Source))
		}
		if result["zone"] != c.Zone {
			errs = append(errs, fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", c.Zone, result["zone"], d.Source))
		}
	}
	return
}

func (c *CreateInstance) validateMachineType(project string, client daisyCompute.Client) []error {
	var errs []error
	if !machineTypeURLRegex.MatchString(c.MachineType) {
		errs = append(errs, fmt.Errorf("can't create instance: bad MachineType: %q", c.MachineType))
	}
	match := machineTypeURLRegex.FindStringSubmatch(c.MachineType)
	if match == nil {
		return append(errs, fmt.Errorf("can't create instance: bad MachineType: %q", c.MachineType))
	}

	result := make(map[string]string)
	for i, name := range machineTypeURLRegex.SubexpNames() {
		if i != 0 {
			result[name] = match[i]
		}
	}
	if result["project"] != "" && result["project"] != c.Project {
		errs = append(errs, fmt.Errorf("cannot create instance in project %q with machineType in project %q: %q", c.Project, result["project"], c.MachineType))
	}
	if result["zone"] != c.Zone {
		errs = append(errs, fmt.Errorf("cannot create instance in zone %q with machineType in zone %q: %q", c.Zone, result["zone"], c.MachineType))
	}

	p := result["project"]
	if p == "" {
		p = project
	}

	if _, err := client.GetMachineType(p, result["zone"], result["machinetype"]); err != nil {
		return append(errs, fmt.Errorf("cannot create instance, bad machineType: %q, error: %v", result["machinetype"], err))
	}

	return errs
}

func (c *CreateInstance) validateNetworks() (errs []error) {
	for _, n := range c.NetworkInterfaces {
		match := networkURLRegex.FindStringSubmatch(n.Network)
		if match == nil {
			errs = append(errs, fmt.Errorf("can't create instance: bad value for NetworkInterface.Network: %q", n.Network))
		} else {
			result := make(map[string]string)
			for i, name := range networkURLRegex.SubexpNames() {
				if i != 0 {
					result[name] = match[i]
				}
			}
			if result["project"] != "" && result["project"] != c.Project {
				errs = append(errs, fmt.Errorf("cannot create instance in project %q with network in project %q: %q", c.Project, result["project"], n.Network))
			}
		}
	}
	return
}

func (c *CreateInstances) validate(ctx context.Context, s *Step) error {
	errs := []error{}
	for _, ci := range *c {
		if !checkName(ci.Name) {
			errs = append(errs, fmt.Errorf("can't create instance %q: bad name", ci.Name))
		}
		if _, err := s.w.ComputeClient.GetProject(ci.Project); err != nil {
			return fmt.Errorf("cannot create disk: bad project: %q, error: %v", ci.Project, err)
		}
		if _, err := s.w.ComputeClient.GetZone(ci.Project, ci.Zone); err != nil {
			return fmt.Errorf("cannot create instance: bad zone: %q, error: %v", ci.Zone, err)
		}

		errs = append(errs, ci.validateDisks(ctx, s)...)
		errs = append(errs, ci.validateMachineType(ci.Project, s.w.ComputeClient)...)
		errs = append(errs, ci.validateNetworks()...)

		// Register creation.
		link := fmt.Sprintf("projects/%s/zones/%s/instances/%s", ci.Project, ci.Zone, ci.Name)
		r := &resource{real: ci.Name, link: link, noCleanup: ci.NoCleanup}
		errs = append(errs, instances[s.w].registerCreation(ci.daisyName, r, s))
	}

	errStrs := []string{}
	for _, e := range errs {
		if e != nil {
			errStrs = append(errStrs, e.Error())
		}
	}
	if len(errStrs) > 0 {
		msg := fmt.Sprintf("%d errors:", len(errStrs))
		for _, s := range errStrs {
			msg += fmt.Sprintf("\n* %s", s)
		}
		return errors.New(msg)
	}
	return nil
}

func (c *CreateInstances) run(ctx context.Context, s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci *CreateInstance) {
			defer wg.Done()

			for _, d := range ci.Disks {
				if diskRes, ok := disks[w].get(d.Source); ok {
					d.Source = diskRes.link
				}
			}

			w.logger.Printf("CreateInstances: creating instance %q.", ci.Name)
			if err := w.ComputeClient.CreateInstance(ci.Project, ci.Zone, &ci.Instance); err != nil {
				e <- err
				return
			}
			go logSerialOutput(ctx, w, ci.Name, 1, 1*time.Second)
		}(ci)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so instances being created now can be deleted.
		wg.Wait()
		return nil
	}
}
