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
	"strconv"
	"sync"

	"google.golang.org/api/compute/v1"
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
	// If set Daisy will use this as the resource name instead generating a name.
	RealName string `json:",omitempty"`

	// The name of the disk as known to the Daisy user.
	daisyName string
	// Deprecated: Use RealName instead.
	ExactName bool
}

// MarshalJSON is a hacky workaround to prevent CreateDisk from using
// compute.Disk's implementation.
func (c *CreateDisk) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
}

func (c *CreateDisks) populate(ctx context.Context, s *Step) dErr {
	for _, cd := range *c {
		cd.daisyName = cd.Name
		if cd.ExactName && cd.RealName == "" {
			cd.RealName = cd.Name
		}
		if cd.RealName != "" {
			cd.Name = cd.RealName
		} else {
			cd.Name = s.w.genName(cd.Name)
		}
		cd.Project = strOr(cd.Project, s.w.Project)
		cd.Zone = strOr(cd.Zone, s.w.Zone)
		cd.Description = strOr(cd.Description, fmt.Sprintf("Disk created by Daisy in workflow %q on behalf of %s.", s.w.Name, s.w.username))
		if cd.SizeGb != "" {
			size, err := strconv.ParseInt(cd.SizeGb, 10, 64)
			if err != nil {
				return errf("cannot parse SizeGb: %s, err: %v", cd.SizeGb, err)
			}
			cd.Disk.SizeGb = size
		}
		if imageURLRgx.MatchString(cd.SourceImage) {
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

func (c *CreateDisks) validate(ctx context.Context, s *Step) dErr {
	for _, cd := range *c {
		if !checkName(cd.Name) {
			return errf("cannot create disk: bad name: %q", cd.Name)
		}
		if exists, err := projectExists(s.w.ComputeClient, cd.Project); err != nil {
			return errf("cannot create disk %q: bad project lookup: %q, error: %v", cd.daisyName, cd.Project, err)
		} else if !exists {
			return errf("cannot create disk %q: project does not exist: %q", cd.daisyName, cd.Project)
		}

		if cd.Zone == "" {
			return errf("cannot create disk %q: no zone provided in step or workflow", cd.daisyName)
		}
		if exists, err := zoneExists(s.w.ComputeClient, cd.Project, cd.Zone); err != nil {
			return errf("cannot create disk %q: bad zone lookup: %q, error: %v", cd.daisyName, cd.Zone, err)
		} else if !exists {
			return errf("cannot create disk %q: zone does not exist: %q", cd.daisyName, cd.Zone)
		}

		if !diskTypeURLRgx.MatchString(cd.Type) {
			return errf("cannot create disk %q: bad disk type: %q", cd.daisyName, cd.Type)
		}

		if cd.SourceImage != "" {
			if _, err := images[s.w].registerUsage(cd.SourceImage, s); err != nil {
				return errf("cannot create disk %q: can't use image %q: %v", cd.daisyName, cd.SourceImage, err)
			}
		} else if cd.Disk.SizeGb == 0 {
			return errf("cannot create disk %q: SizeGb and SourceImage not set", cd.daisyName)
		}

		// Register creation.
		link := fmt.Sprintf("projects/%s/zones/%s/disks/%s", cd.Project, cd.Zone, cd.Name)
		r := &resource{real: cd.Name, link: link, noCleanup: cd.NoCleanup}
		if err := disks[s.w].registerCreation(cd.daisyName, r, s, false); err != nil {
			return err
		}
	}
	return nil
}

func (c *CreateDisks) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)
	for _, cd := range *c {
		wg.Add(1)
		go func(cd *CreateDisk) {
			defer wg.Done()

			// Get the source image link if using a source image.
			if cd.SourceImage != "" {
				image, _ := images[w].get(cd.SourceImage)
				cd.SourceImage = image.link
			}

			w.Logger.Printf("CreateDisks: creating disk %q.", cd.Name)
			if err := w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
				e <- newErr(err)
				return
			}
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
