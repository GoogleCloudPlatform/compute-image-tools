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
	"strings"
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
	SourceImage string `json:"source_image"`
	// Size of this disk.
	SizeGB string
	// Is this disk PD-SSD.
	SSD bool
}

func (c *CreateDisks) validate() error {
	for _, cd := range *c {
		// Image checking.
		if cd.SourceImage != "" && !imageExists(cd.SourceImage) {
			return fmt.Errorf("cannot create disk: image not found: %s", cd.SourceImage)
		}

		_, err := strconv.ParseInt(cd.SizeGB, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse SizeGB: %s, err: %v", cd.SizeGB, err)
		}

		// Try adding disk name.
		if err := diskNames.add(cd.Name); err != nil {
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
			name := namer(cd.Name, w.Name, w.id)
			// If cd.SourceImage does not contain a '/' assume it's referencing a Workflow image.
			if !strings.Contains(cd.SourceImage, "/") {
				cd.SourceImage = w.getCreatedImage(cd.SourceImage)
			}
			size, err := strconv.ParseInt(cd.SizeGB, 10, 64)
			if err != nil {
				e <- err
				return
			}
			d, err := w.ComputeClient.CreateDisk(name, w.Project, w.Zone, cd.SourceImage, size, cd.SSD)
			if err != nil {
				e <- err
				return
			}
			w.addCreatedDisk(name, d.SelfLink)
		}(cd)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Ctx.Done():
		// Wait so disks being created now can be deleted.
		wg.Wait()
		return nil
	}
}
