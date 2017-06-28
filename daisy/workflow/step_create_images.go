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
	"context"
	"errors"
	"fmt"
	"sync"
	"encoding/json"

	compute "google.golang.org/api/compute/v1"
)

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages []*CreateImage

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
	daisyName string
}

// MarshalJSON is a hacky workaround to prevent CreateImage from using
// compute.Image's implementation.
func (c *CreateImage) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

// populate preprocesses fields: Name, Project, Description, SourceDisk, RawDisk, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateImages) populate(ctx context.Context, s *Step) error {
	for _, ci := range *c {
		// Prepare field values: name, Name, RawDisk.Source, Description
		ci.daisyName = ci.Name
		if !ci.ExactName {
			ci.Name = s.w.genName(ci.daisyName)
		}
		ci.Project = strOr(ci.Project, s.w.Project)
		ci.Description = strOr(ci.Description, fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))

		if diskURLRegex.MatchString(ci.SourceDisk) {
			ci.SourceDisk = extendPartialURL(ci.SourceDisk, ci.Project)
		}

		if ci.RawDisk != nil {
			if s.w.sourceExists(ci.RawDisk.Source) {
				ci.RawDisk.Source = s.w.getSourceGCSAPIPath(ci.RawDisk.Source)
			} else if p, err := getGCSAPIPath(ci.RawDisk.Source); err == nil {
				ci.RawDisk.Source = p
			} else {
				return fmt.Errorf("bad value for RawDisk.Source: %q", ci.RawDisk.Source)
			}
		}
	}
	return nil
}

func (c *CreateImages) validate(ctx context.Context, s *Step) error {
	if err := c.populate(ctx, s); err != nil {
		return err
	}
	for _, ci := range *c {
		if !checkName(ci.Name) {
			return fmt.Errorf("can't create image: bad name: %q", ci.Name)
		}

		// Source disk checking.
		if !xor(ci.SourceDisk == "", ci.RawDisk == nil) {
			return errors.New("must provide either SourceDisk or RawDisk, exclusively")
		}

		if ci.SourceDisk != "" {
			if !diskValid(s.w, ci.SourceDisk) {
				return fmt.Errorf("cannot create image: disk not found: %s", ci.SourceDisk)
			}
		}

		// Try adding image name.
		if err := validatedImages.add(s.w, ci.daisyName); err != nil {
			return fmt.Errorf("error adding image: %s", err)
		}
	}

	return nil
}

func (c *CreateImages) run(ctx context.Context, s *Step) error {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan error)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci *CreateImage) {
			defer wg.Done()

			project := strOr(ci.Project, w.Project)

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
			images[w].add(ci.daisyName, &resource{real: ci.Name, link: ci.SelfLink, noCleanup: ci.NoCleanup})
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
