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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"path"
	"strings"
	"sync"
	"time"
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

func logSerialOutput(w *Workflow, name string, port int64) {
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.logger.Printf("CreateInstances: streaming instance %q serial port %d output to gs://%s/%s.", name, port, w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	var errs int
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-w.Ctx.Done():
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
					continue
				}
				w.logger.Printf("CreateInstances: instance %q: error getting serial port: %v", name, err)
				return
			}
			start = resp.Next
			buf.WriteString(resp.Contents)
			wc := w.StorageClient.Bucket(w.bucket).Object(logsObj).NewWriter(w.Ctx)
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

func (c *CreateInstance) processDisks(w *Workflow) error {
	if len(c.Disks) == 0 {
		return errors.New("cannot create instance: no disks provided")
	}

	for i, d := range c.Disks {
		d.Boot = i == 0
		if !diskValid(w, d.Source) {
			return fmt.Errorf("cannot create instance: disk not found: %s", d.Source)
		}
		d.Mode = strOr(d.Mode, "READ_WRITE")
		if !strIn(d.Mode, []string{"READ_ONLY", "READ_WRITE"}) {
			return fmt.Errorf("cannot create instance: bad disk mode: %q", d.Mode)
		}

		// Disk is a partial URL, ensure disk is in the same project and zone.
		match := diskURLRegex.FindStringSubmatch(d.Source)
		if match == nil {
			continue
		}
		result := make(map[string]string)
		for i, name := range diskURLRegex.SubexpNames() {
			if i != 0 {
				result[name] = match[i]
			}
		}

		if result["project"] != "" && result["project"] != c.Project {
			return fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", c.Project, result["project"], d.Source)
		}
		if result["zone"] != c.Zone {
			return fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", c.Zone, result["zone"], d.Source)
		}
	}
	return nil
}

func (c *CreateInstance) processMachineType() error {
	c.MachineType = strOr(c.MachineType, "n1-standard-1")
	mt := c.MachineType
	if !strings.Contains(c.MachineType, "/") {
		c.MachineType = fmt.Sprintf("zones/%s/machineTypes/%s", c.Zone, c.MachineType)
	}
	if !machineTypeURLRegex.MatchString(c.MachineType) {
		return fmt.Errorf("cannot create instance: bad machine type %q", mt)
	}
	return nil
}

func (c *CreateInstance) processMetadata(w *Workflow) error {
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
			return fmt.Errorf("cannot create instance: file not found: %s", c.StartupScript)
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

func (c *CreateInstance) processNetworks() error {
	defaultAcs := []*compute.AccessConfig{{Type: "ONE_TO_ONE_NAT"}}
	if c.NetworkInterfaces == nil {
		c.NetworkInterfaces = []*compute.NetworkInterface{{AccessConfigs: defaultAcs, Network: "global/networks/default"}}
	} else {
		for _, n := range c.NetworkInterfaces {
			if n.AccessConfigs == nil {
				n.AccessConfigs = defaultAcs
			}
			if !networkURLRegex.MatchString(n.Network) {
				if !rfc1035Rgx.MatchString(n.Network) {
					return fmt.Errorf("%q is not a valid network", n.Network)
				}
				n.Network = path.Join("global/networks", n.Network)
			}
		}
	}
	return nil
}

func (c *CreateInstance) processScopes() error {
	if len(c.Scopes) == 0 {
		c.Scopes = append(c.Scopes, "https://www.googleapis.com/auth/devstorage.read_only")
	}
	if c.ServiceAccounts == nil {
		c.ServiceAccounts = []*compute.ServiceAccount{{Email: "default", Scopes: c.Scopes}}
	}
	return nil
}

func (c *CreateInstances) validate(s *Step) error {
	w := s.w

	errs := []error{}
	for _, ci := range *c {
		// General fields preprocessing.
		ci.daisyName = ci.Name
		if !ci.ExactName {
			ci.Name = w.genName(ci.Name)
		}
		ci.Project = strOr(ci.Project, w.Project)
		ci.Zone = strOr(ci.Zone, w.Zone)
		ci.Description = strOr(ci.Description, fmt.Sprintf("Instance created by Daisy in workflow %q on behalf of %s.", w.Name, w.username))

		errs = append(errs, ci.processDisks(w))
		errs = append(errs, ci.processMachineType())
		errs = append(errs, ci.processMetadata(w))
		errs = append(errs, ci.processNetworks())
		errs = append(errs, ci.processScopes())

		// Try adding instance name.
		errs = append(errs, validatedInstances.add(w, ci.daisyName))
	}

	var multiErr *multierror.Error
	if len(errs) > 0 {
		multiErr = multierror.Append(errs[0], errs[1:]...)
	}
	return multiErr.ErrorOrNil()
}

func (c *CreateInstances) run(s *Step) error {
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
			go logSerialOutput(w, ci.Name, 1)
			instances[w].add(ci.daisyName, &resource{ci.daisyName, ci.Name, ci.SelfLink, ci.NoCleanup, false})
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
