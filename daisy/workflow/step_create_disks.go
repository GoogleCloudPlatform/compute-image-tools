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
	"strconv"
	"sync"

	"encoding/json"
	compute "google.golang.org/api/compute/v1"
)

// CreateDisks is a Daisy CreateDisks workflow step.
type CreateDisks []*CreateDisk

// CreateDisk describes a GCE disk.
type CreateDisk struct {
	compute.Disk

	// Size of this disk.
	SizeGb string `json:"sizeGb,omitempty"`
	// Zone to create the instance in, overrides workflow Zone.
	Zone string `json:",omitempty"`
	// Project to create the instance in, overrides workflow Project.
	Project string `json:",omitempty"`
	// Should this resource be cleaned up after the workflow?
	NoCleanup bool
	// Should we use the user-provided reference name as the actual
	// resource name?
	ExactName bool

	// The name of the disk as known internally to Daisy.
	daisyName string
}

// MarshalJSON is a hacky workaround to prevent CreateDisk from using
// compute.Disk's implementation.
func (c *CreateDisk) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

func (c *CreateDisks) populate(s *Step) error {
	for _, cd := range *c {
		cd.daisyName = cd.Name
		if !cd.ExactName {
			cd.Name = s.w.genName(cd.daisyName)
		}
		cd.Project = strOr(cd.Project, s.w.Project)
		cd.Zone = strOr(cd.Zone, s.w.Zone)
		cd.Description = strOr(cd.Description, fmt.Sprintf("Disk created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))
		if cd.SizeGb != "" {
			size, err := strconv.ParseInt(cd.SizeGb, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot parse SizeGb: %s, err: %v", cd.SizeGb, err)
			}
			cd.Disk.SizeGb = size
		}
		if imageURLRegex.MatchString(cd.SourceImage) {
			cd.SourceImage = extendPartialURL(cd.SourceImage, cd.Project)
		}
		if cd.Type == "" {
			cd.Type = fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", cd.Project, cd.Zone)
		} else if diskTypeURLRgx.MatchString(cd.Type) {
			cd.Type = extendPartialURL(cd.Type, cd.Project)
		} else {
			cd.Type = fmt.Sprintf("projects/%s/zones/%s/diskTypes/%s", cd.Project, cd.Zone, cd.Type)
		}
	}
	return nil
}

func (c *CreateDisks) validate(s *Step) error {
	for _, cd := range *c {
		if !checkName(cd.Name) {
			return fmt.Errorf("cannot create disk: invalid name: %q", cd.Name)
		}
		if !rfc1035Rgx.MatchString(cd.Project) {
			return fmt.Errorf("cannot create disk: invalid project: %q", cd.Project)
		}
		if !rfc1035Rgx.MatchString(cd.Zone) {
			return fmt.Errorf("cannot create disk: invalid zone: %q", cd.Zone)
		}
		if !diskTypeURLRgx.MatchString(cd.Type) {
			return fmt.Errorf("cannot create disk: invalid disk type: %q", cd.Type)
		}

		if cd.SourceImage != "" {
			if !imageValid(s.w, cd.SourceImage) {
				return fmt.Errorf("cannot create disk: image not found: %q", cd.SourceImage)
			}
		} else if cd.SizeGb == "" {
			return errors.New("cannot create disk: SizeGb and SourceImage not set")
		}

		// Try adding disk name.
		if err := validatedDisks.add(s.w, cd.daisyName); err != nil {
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
		go func(cd *CreateDisk) {
			defer wg.Done()

			// Get the source image link if using a source image.
			if cd.SourceImage != "" && !imageURLRegex.MatchString(cd.SourceImage) {
				image, _ := images[w].get(cd.SourceImage)
				cd.SourceImage = image.link
			}

			w.logger.Printf("CreateDisks: creating disk %q.", cd.Name)
			if err := w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
				e <- err
				return
			}
			disks[w].add(cd.daisyName, &resource{real: cd.Name, link: cd.SelfLink, noCleanup: cd.NoCleanup})
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
