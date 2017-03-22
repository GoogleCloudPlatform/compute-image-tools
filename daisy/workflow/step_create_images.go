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
	"sync"
)

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages []CreateImage

// CreateImage creates a GCE image in a project.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type CreateImage struct {
	// Name if the image.
	Name string
	// Project to import image into. If this is unset Workflow.Project is used.
	Project string
	// Image family
	Family string
	// Image licenses
	Licenses []string
	// A list of features to enable on the guest OS.
	// https://godoc.org/google.golang.org/api/compute/v1#GuestOsFeature
	GuestOsFeatures []string `json:"guest_os_features"`
	// Only one of these source types should be specified.
	SourceDisk string `json:"source_disk"`
	SourceFile string `json:"source_file"`
}

func (c *CreateImages) validate(w *Workflow) error {
	for _, ci := range *c {
		// File/Disk checking.
		if !xor(ci.SourceDisk == "", ci.SourceFile == "") {
			return errors.New("must provide either Disk or File, exclusively")
		}
		if ci.SourceDisk != "" && !isLink(ci.SourceDisk) && !diskValid(w, ci.SourceDisk) {
			return fmt.Errorf("cannot create image: disk not found: %s", ci.SourceDisk)
		}
		if ci.SourceFile != "" && !sourceExists(ci.SourceFile) {
			return fmt.Errorf("cannot create image: file not found: %s", ci.SourceFile)
		}

		// Project checking.
		if !projectExists(ci.Project) {
			return fmt.Errorf("cannot create image: project not found: %s", ci.Project)
		}

		// Try adding image name.
		if err := validatedImages.add(w, ci.Name); err != nil {
			return fmt.Errorf("error adding image: %s", err)
		}
	}

	return nil
}

func (c *CreateImages) run(w *Workflow) error {
	var wg sync.WaitGroup
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci CreateImage) {
			defer wg.Done()
			name := w.ephemeralName(ci.Name)
			var diskLink string
			if ci.SourceDisk != "" {
				// Using source disk case.
				diskLink = resolveLink(ci.SourceDisk, w.diskRefs)
				if diskLink == "" {
					e <- fmt.Errorf("unresolved disk %q", ci.SourceDisk)
					return
				}
			}
			i, err := w.ComputeClient.CreateImage(name, w.Project, diskLink, ci.SourceFile, ci.Family, ci.Licenses, ci.GuestOsFeatures)
			if err != nil {
				e <- err
				return
			}
			w.imageRefs.add(ci.Name, &resource{ci.Name, name, i.SelfLink, false})
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
		// Wait so images being created now will complete.
		wg.Wait()
		return nil
	}
}
