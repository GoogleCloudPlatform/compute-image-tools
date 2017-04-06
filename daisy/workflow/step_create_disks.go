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
	"fmt"
	"strconv"
	"sync"
)

// CreateDisks is a Daisy CreateDisks workflow step.
type CreateDisks []CreateDisk

// CreateDisk describes a GCE disk.
type CreateDisk struct {
	// Name of the disk.
	Name string
	// SourceImage to use during disk creation. Leave blank for a blank disk.
	// See https://godoc.org/google.golang.org/api/compute/v1#Disk.
	SourceImage string
	// Size of this disk.
	SizeGB string
	// Type of disk, pd-standard (default) or pd-ssd.
	Type string
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual resource name?
	ExactName bool
}

func (c *CreateDisks) validate(w *Workflow) error {
	for _, cd := range *c {
		// Image checking.
		if cd.SourceImage != "" && !imageValid(w, cd.SourceImage) {
			return fmt.Errorf("cannot create disk: image not found: %s", cd.SourceImage)
		}

		if _, err := strconv.ParseInt(cd.SizeGB, 10, 64); cd.SizeGB != "" && err != nil {
			return fmt.Errorf("cannot parse SizeGB: %s, err: %v", cd.SizeGB, err)
		}

		// No SizeGB set when not supplying SourceImage.
		if cd.SizeGB == "" && cd.SourceImage == "" {
			return fmt.Errorf("cannot create disk: SizeGB and SourceImage not set: %s", cd.SourceImage)
		}

		// Try adding disk name.
		if err := validatedDisks.add(w, cd.Name); err != nil {
			return fmt.Errorf("error adding disk: %s", err)
		}
	}

	return nil
}

func (c *CreateDisks) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)
	for _, cd := range *c {
		wg.Add(1)
		go func(cd CreateDisk) {
			defer wg.Done()
			name := cd.Name
			if !cd.ExactName {
				name = w.genName(cd.Name)
			}

			// Get the source image link.
			var imageLink string
			var image *resource
			var err error
			if cd.SourceImage == "" || isLink(cd.SourceImage) {
				imageLink = cd.SourceImage
			} else if image, err = w.getImage(cd.SourceImage); err == nil {
				imageLink = image.link
			} else {
				e <- err
				return
			}

			size, err := strconv.ParseInt(cd.SizeGB, 10, 64)
			if cd.SizeGB != "" && err != nil {
				e <- err
				return
			}

			w.logger.Printf("CreateDisks: creating disk %q.", name)
			d, err := w.ComputeClient.CreateDisk(name, w.Project, w.Zone, imageLink, size, cd.Type)
			if err != nil {
				e <- err
				return
			}
			w.diskRefs.add(cd.Name, &resource{cd.Name, name, d.SelfLink, cd.NoCleanup})
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
