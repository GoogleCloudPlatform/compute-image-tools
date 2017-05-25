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
	"sync"

	compute "google.golang.org/api/compute/v1"
)

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages []CreateImage

// CreateImage creates a GCE image in a project.
// Supported sources are a GCE disk or a RAW image listed in Workflow.Sources.
type CreateImage struct {
	compute.Image

	// Project to import image into. If this is unset Workflow.Project is
	// used.
	Project string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual
	// resource name?
	ExactName bool

	// The name of the disk as known internally to Daisy.
	name string
}

func (c *CreateImages) validate(s *Step) error {
	w := s.w
	for _, ci := range *c {
		// Source disk checking.
		if !xor(ci.SourceDisk == "", ci.RawDisk == nil) {
			return errors.New("must provide either SourceDisk or RawDisk, exclusively")
		}

		// Prepare field values: name, Name, RawDisk.Source, Description
		ci.name = ci.Name
		if !ci.ExactName {
			ci.Name = w.genName(ci.name)
		}
		if ci.SourceDisk != "" {
			if !diskValid(w, ci.SourceDisk) {
				return fmt.Errorf("cannot create image: disk not found: %s", ci.SourceDisk)
			}
		} else if w.sourceExists(ci.RawDisk.Source) {
			ci.RawDisk.Source = fmt.Sprintf("https://storage.cloud.google.com/%s", path.Join(w.bucket, w.sourcesPath, ci.RawDisk.Source))
		} else if b, o, err := splitGCSPath(ci.RawDisk.Source); err == nil {
			ci.RawDisk.Source = fmt.Sprintf("https://storage.cloud.google.com/%s", path.Join(b, o))
		} else {
			return fmt.Errorf("cannot create image: file not in sources or valid GCS path: %s", ci.RawDisk.Source)
		}
		ci.Description = stringOr(ci.Description, fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", w.Name, w.username))

		// Project checking.
		if ci.Project != "" && !projectExists(ci.Project) {
			return fmt.Errorf("cannot create image: project not found: %s", ci.Project)
		}

		// Try adding image name.
		if err := validatedImages.add(w, ci.name); err != nil {
			return fmt.Errorf("error adding image: %s", err)
		}
	}

	return nil
}

func (c *CreateImages) run(s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci CreateImage) {
			defer wg.Done()

			project := stringOr(ci.Project, w.Project)

			// Get source disk link, if applicable.  TODO(crunkleton): Move to validate after validation refactor.
			if ci.SourceDisk != "" {
				if disk, ok := disks[w].get(ci.SourceDisk); ok {
					ci.SourceDisk = disk.link
				} else if !diskURLRegex.MatchString(ci.SourceDisk) {
					e <- fmt.Errorf("invalid or missing reference to SourceDisk %q", ci.SourceDisk)
					return
				}
			}

			w.logger.Printf("CreateImages: creating image %q.", ci.Name)
			err := w.ComputeClient.CreateImage(project, &ci.Image)
			if err != nil {
				e <- err
				return
			}
			images[w].add(ci.name, &resource{ci.name, ci.Name, ci.SelfLink, ci.NoCleanup, false})
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
		// Wait so images being created now will complete.
		wg.Wait()
		return nil
	}
}
