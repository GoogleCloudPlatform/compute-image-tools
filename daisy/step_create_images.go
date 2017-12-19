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

package daisy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
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
	// If set Daisy will use this as the resource name instead generating a name.
	RealName string `json:",omitempty"`
	// Should an existing image of the same name be deleted, defaults to false
	// which will fail validation.
	OverWrite bool

	// The name of the disk as known to the Daisy user.
	daisyName string
	// Deprecated: Use RealName instead.
	ExactName bool
}

// MarshalJSON is a hacky workaround to prevent CreateImage from using
// compute.Image's implementation.
func (c *CreateImage) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

// populate preprocesses fields: Name, Project, Description, SourceDisk, RawDisk, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateImages) populate(ctx context.Context, s *Step) dErr {
	for _, ci := range *c {
		// Prepare field values: name, Name, RawDisk.Source, Description
		ci.daisyName = ci.Name
		if ci.ExactName && ci.RealName == "" {
			ci.RealName = ci.Name
		}
		if ci.RealName != "" {
			ci.Name = ci.RealName
		} else {
			ci.Name = s.w.genName(ci.Name)
		}
		ci.Project = strOr(ci.Project, s.w.Project)
		ci.Description = strOr(ci.Description, fmt.Sprintf("Image created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))

		if diskURLRgx.MatchString(ci.SourceDisk) {
			ci.SourceDisk = extendPartialURL(ci.SourceDisk, ci.Project)
		}

		if imageURLRgx.MatchString(ci.SourceImage) {
			ci.SourceImage = extendPartialURL(ci.SourceImage, ci.Project)
		}

		if ci.RawDisk != nil {
			if s.w.sourceExists(ci.RawDisk.Source) {
				ci.RawDisk.Source = s.w.getSourceGCSAPIPath(ci.RawDisk.Source)
			} else if p, err := getGCSAPIPath(ci.RawDisk.Source); err == nil {
				ci.RawDisk.Source = p
			} else {
				return errf("bad value for RawDisk.Source: %q", ci.RawDisk.Source)
			}
		}
	}
	return nil
}

func (c *CreateImages) validate(ctx context.Context, s *Step) dErr {
	for _, ci := range *c {
		if !checkName(ci.Name) {
			return errf("can't create image: bad name: %q", ci.Name)
		}

		// Project checking.
		if exists, err := projectExists(s.w.ComputeClient, ci.Project); err != nil {
			return errf("cannot create image %q: bad project lookup: %q, error: %v", ci.daisyName, ci.Project, err)
		} else if !exists {
			return errf("cannot create image %q: project does not exist: %q", ci.daisyName, ci.Project)
		}

		if !xor(!xor(ci.SourceDisk == "", ci.SourceImage == ""), ci.RawDisk == nil) {
			return errf("cannot create image %q: must provide either SourceImage, SourceDisk or RawDisk, exclusively", ci.daisyName)
		}

		// Source disk checking.
		if ci.SourceDisk != "" {
			if _, err := disks[s.w].registerUsage(ci.SourceDisk, s); err != nil {
				return newErr(err)
			}
		}

		// Source image checking.
		if ci.SourceImage != "" {
			if _, err := images[s.w].registerUsage(ci.SourceImage, s); err != nil {
				return newErr(err)
			}
		}

		// License checking.
		for _, l := range ci.Licenses {
			result := namedSubexp(licenseURLRegex, l)
			if exists, err := licenseExists(s.w.ComputeClient, result["project"], result["license"]); err != nil {
				return errf("cannot create image %q: bad license lookup: %q, error: %v", ci.daisyName, l, err)
			} else if !exists {
				return errf("cannot create image %q: license does not exist: %q", ci.daisyName, l)
			}
		}

		// Register image creation.
		link := fmt.Sprintf("projects/%s/global/images/%s", ci.Project, ci.Name)
		r := &resource{real: ci.Name, link: link, noCleanup: ci.NoCleanup}
		if err := images[s.w].registerCreation(ci.daisyName, r, s, ci.OverWrite); err != nil {
			return err
		}
	}

	return nil
}

func (c *CreateImages) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)
	for _, ci := range *c {
		wg.Add(1)
		go func(ci *CreateImage) {
			defer wg.Done()
			// Get source disk link if SourceDisk is a daisy reference to a disk.
			if d, ok := disks[w].get(ci.SourceDisk); ok {
				ci.SourceDisk = d.link
			}

			// Delete existing if OverWrite is true.
			if ci.OverWrite {
				// Just try to delete it, a 404 here indicates the image doesn't exist.
				if err := w.ComputeClient.DeleteImage(ci.Project, ci.Name); err != nil {
					if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
						e <- errf("error deleting existing image: %v", err)
						return
					}
				}
			}

			w.Logger.Printf("CreateImages: creating image %q.", ci.Name)
			if err := w.ComputeClient.CreateImage(ci.Project, &ci.Image); err != nil {
				e <- newErr(err)
				return
			}
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
		// Wait so images being created now will complete before we try to clean them up.
		wg.Wait()
		return nil
	}
}
