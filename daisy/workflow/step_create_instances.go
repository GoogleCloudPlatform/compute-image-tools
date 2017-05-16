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
	"errors"
	"fmt"
	"path"
	"sync"
	"time"
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances []CreateInstance

// CreateInstance creates a GCE instance. Output of serial port 1 will be
// streamed to the daisy logs directory.
type CreateInstance struct {
	// Name of the instance.
	Name string
	// Disks to attach to the instance, must match a disk created in a
	// previous step.
	// The first disk gets set as the boot disk. At least one disk must be
	// listed.
	AttachedDisks []string
	MachineType   string `json:",omitempty"`
	// StartupScript is the Sources path to a startup script to use in this step.
	// This will be automatically mapped to the appropriate metadata key.
	StartupScript string `json:",omitempty"`
	// Additional metadata to set for the instance.
	Metadata map[string]string `json:",omitempty"`
	// OAuth2 scopes to give the instance. If non are specified
	// https://www.googleapis.com/auth/devstorage.read_only will be added.
	Scopes []string `json:",omitempty"`
	// Optional description of the resource, if not specified Daisy will
	// create one with the name of the project.
	Description string `json:",omitempty"`
	// Zone to create the instance in, overrides workflow Zone.
	Zone string `json:",omitempty"`
	// Project to create the instance in, overrides workflow Project.
	Project string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual resource name?
	ExactName bool
}

func logSerialOutput(w *Workflow, name string, port int64) {
	logsObj := path.Join(w.logsPath, fmt.Sprintf("%s-serial-port%d.log", name, port))
	w.logger.Printf("CreateInstances: streaming instance %q serial port %d output to gs://%s/%s.", name, port, w.bucket, logsObj)
	var start int64
	var buf bytes.Buffer
	tick := time.Tick(1 * time.Second)
	for {
		select {
		case <-w.Ctx.Done():
			return
		case <-tick:
			resp, err := w.ComputeClient.GetSerialPortOutput(w.Project, w.Zone, name, port, start)
			if err != nil {
				stopped, sErr := w.ComputeClient.InstanceStopped(w.Project, w.Zone, name)
				if stopped && sErr == nil {
					return
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
				w.logger.Printf("CreateInstances: instance %q: error writing log to GCS: %v", name, err)
				return
			}
		}
	}
}

func (c *CreateInstances) validate(w *Workflow) error {
	for _, ci := range *c {
		// Disk checking.
		if len(ci.AttachedDisks) == 0 {
			return errors.New("cannot create instance: no disks provided")
		}
		for _, d := range ci.AttachedDisks {
			if !diskValid(w, d) {
				return fmt.Errorf("cannot create instance: disk not found: %s", d)
			}
			// Ensure disk is in the same project and zone.
			match := diskURLRegex.FindStringSubmatch(d)
			if match == nil {
				continue
			}
			result := make(map[string]string)
			for i, name := range diskURLRegex.SubexpNames() {
				if i != 0 {
					result[name] = match[i]
				}
			}

			project := w.Project
			if ci.Project != "" {
				project = ci.Project
			}
			zone := w.Zone
			if ci.Zone != "" {
				zone = ci.Zone
			}

			if result["project"] != "" && result["project"] != project {
				return fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", project, result["project"], d)
			}
			if result["zone"] != zone {
				return fmt.Errorf("cannot create instance in project %q with disk in project %q: %q", zone, result["zone"], d)
			}
		}

		// Startup script checking.
		if ci.StartupScript != "" && !w.sourceExists(ci.StartupScript) {
			return fmt.Errorf("cannot create instance: file not found: %s", ci.StartupScript)
		}

		// Try adding instance name.
		if err := validatedInstances.add(w, ci.Name); err != nil {
			return fmt.Errorf("error adding instance: %s", err)
		}
	}

	return nil
}

func (c *CreateInstances) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci CreateInstance) {
			defer wg.Done()
			name := ci.Name
			if !ci.ExactName {
				name = w.genName(ci.Name)
			}

			zone := w.Zone
			if ci.Zone != "" {
				zone = ci.Zone
			}

			project := w.Project
			if ci.Project != "" {
				project = ci.Project
			}

			inst, err := w.ComputeClient.NewInstance(name, project, zone, ci.MachineType)
			if err != nil {
				e <- err
				return
			}
			inst.Scopes = ci.Scopes
			description := ci.Description
			if description == "" {
				description = fmt.Sprintf("Instance created by Daisy in workflow %q on behalf of %s.", w.Name, w.username)
			}
			inst.Description = description

			for i, sourceDisk := range ci.AttachedDisks {
				var disk *resource
				var err error
				if diskURLRegex.MatchString(sourceDisk) {
					// Real link.
					inst.AddPD("", sourceDisk, false, i == 0)
				} else if disk, err = w.getDisk(sourceDisk); err == nil {
					// Reference.
					inst.AddPD(disk.name, disk.link, false, i == 0)
				} else {
					e <- err
					return
				}
			}
			if ci.StartupScript != "" {
				script := "gs://" + path.Join(w.bucket, w.sourcesPath, ci.StartupScript)
				inst.AddMetadata(map[string]string{"startup-script-url": script, "windows-startup-script-url": script})
			}
			inst.AddMetadata(ci.Metadata)
			// Add standard Daisy metadata.
			md := map[string]string{
				"daisy-sources-path": "gs://" + path.Join(w.bucket, w.sourcesPath),
				"daisy-logs-path":    "gs://" + path.Join(w.bucket, w.logsPath),
				"daisy-outs-path":    "gs://" + path.Join(w.bucket, w.outsPath),
			}
			inst.AddMetadata(md)
			inst.AddNetworkInterface("global/networks/default")

			w.logger.Printf("CreateInstances: creating instance %q.", name)
			i, err := inst.Insert()
			if err != nil {
				e <- err
				return
			}
			go logSerialOutput(w, name, 1)
			w.instanceRefs.add(ci.Name, &resource{ci.Name, name, i.SelfLink, ci.NoCleanup})
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
