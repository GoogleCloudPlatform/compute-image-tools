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
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sync"
)

// CreateInstances is a Daisy CreateInstances workflow step.
type CreateInstances []CreateInstance

// CreateInstance describes a GCE instance.
type CreateInstance struct {
	// Name of the instance.
	Name string
	// Disks to attach to the instance, must match a disk created in a previous step.
	// First one gets set as boot disk. At least one disk must be listed.
	AttachedDisks []string `json:"attached_disks"`
	MachineType   string   `json:"machine_type"`
	// StartupScript is the local path to a startup script to use in this step.
	// This will be automatically mapped to the appropriate metadata key.
	StartupScript string `json:"startup_script"`
	// Additional metadata to set for the instance.
	Metadata map[string]string
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool `json:"no_cleanup"`
	// Should we use the user-provided reference name as the actual resource name?
	ExactName bool `json:"exact_name"`
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
		}

		// Startup script checking.
		if !sourceExists(ci.StartupScript) {
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

			inst, err := w.ComputeClient.NewInstance(name, w.Project, w.Zone, ci.MachineType)
			if err != nil {
				e <- err
				return
			}

			for i, sourceDisk := range ci.AttachedDisks {
				if isLink(sourceDisk) {
					// Real link.
					inst.AddPD("", sourceDisk, false, i == 0)
				} else if r, ok := w.diskRefs.get(sourceDisk); ok {
					// Reference.
					inst.AddPD(r.name, r.link, false, i == 0)
				} else {
					e <- fmt.Errorf("unresolved disk %q", sourceDisk)
					return
				}
			}
			if ci.StartupScript != "" {
				var startup string
				switch filepath.Ext(ci.StartupScript) {
				case ".ps1", ".bat", ".cmd":
					startup = "windows-startup-script-url"
				default:
					startup = "startup-script-url"
				}
				inst.AddMetadata(map[string]string{startup: path.Join(w.sourcesPath, ci.StartupScript)})
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

			i, err := inst.Insert()
			if err != nil {
				e <- err
				return
			}
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
	case <-w.Ctx.Done():
		// Wait so instances being created now can be deleted.
		wg.Wait()
		return nil
	}
}
