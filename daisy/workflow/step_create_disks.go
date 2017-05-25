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
	"strings"
	"sync"

	compute "google.golang.org/api/compute/v1"
)

// CreateDisks is a Daisy CreateDisks workflow step.
type CreateDisks []CreateDisk

// CreateDisk describes a GCE disk.
type CreateDisk struct {
	compute.Disk

	Zone string `json:",omitempty"`
	// Project to create the instance in, overrides workflow Project.
	Project string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual
	// resource name?
	ExactName bool

	// The name of the disk as known internally to Daisy.
	name string
}

func (c *CreateDisks) validate(s *Step) error {
	w := s.w
	for _, cd := range *c {
		// Image checking.
		if cd.SourceImage != "" && !imageValid(w, cd.SourceImage) {
			return fmt.Errorf("cannot create disk: image not found: %s", cd.SourceImage)
		}

		// No SizeGB set when not supplying SourceImage.
		if cd.SizeGb == 0 && cd.SourceImage == "" {
			return errors.New("cannot create disk: SizeGb and SourceImage not set")
		}

		// Prepare field values: Disk.Name, Disk.Type, name, Project, Zone
		cd.name = cd.Name
		if !cd.ExactName {
			cd.Name = w.genName(cd.name)
		}
		cd.Project = stringOr(cd.Project, w.Project)
		cd.Zone = stringOr(cd.Zone, w.Zone)
		dt := fmt.Sprintf("zones/%s/diskTypes/pd-standard", cd.Zone)
		if cd.Type != "" {
			if strings.Contains(cd.Type, "/") {
				dt = cd.Type
			} else {
				dt = fmt.Sprintf("zones/%s/diskTypes/%s", cd.Zone, cd.Type)
			}
		}
		cd.Type = dt
		cd.Description = stringOr(cd.Description, fmt.Sprintf("Disk created by Daisy in workflow %q on behalf of %s.", w.Name, w.username))

		// Try adding disk name.
		if err := validatedDisks.add(w, cd.name); err != nil {
			return fmt.Errorf("error adding disk: %s", err)
		}
	}

	return nil
}

func (c *CreateDisks) run(s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, cd := range *c {
		wg.Add(1)
		go func(cd CreateDisk) {
			defer wg.Done()

			// Get the source image link.  TODO(crunkleton): Move to validate after validation refactor.
			if cd.SourceImage != "" && !imageURLRegex.MatchString(cd.SourceImage) {
				if image, ok := images[w].get(cd.SourceImage); ok {
					cd.SourceImage = image.link
				} else {
					e <- fmt.Errorf("invalid or missing reference to SourceImage %q", cd.SourceImage)
					return
				}
			}

			w.logger.Printf("CreateDisks: creating disk %q.", cd.Name)
			if err := w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
				e <- err
				return
			}
			disks[w].add(cd.name, &resource{cd.name, cd.Name, cd.SelfLink, cd.NoCleanup, false})
		}(cd)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so disks being created now can be deleted.
		wg.Wait()
		return nil
	}
}
