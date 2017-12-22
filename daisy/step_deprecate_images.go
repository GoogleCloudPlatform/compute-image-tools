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
	"fmt"
	"sync"

	"google.golang.org/api/compute/v1"
)

// DeprecateImages is a Daisy DeprecateImage workflow step.
type DeprecateImages []*DeprecateImage

// DeprecateImage sets the deprecation status on a GCE image.
type DeprecateImage struct {
	// Image to set deprecation status on.
	Image string
	// DeprecationStatus to set for image.
	DeprecationStatus compute.DeprecationStatus
	// Project image is in, overrides workflow Project.
	Project string `json:",omitempty"`
}

func (d *DeprecateImages) populate(ctx context.Context, s *Step) dErr {
	for _, di := range *d {
		di.Project = strOr(di.Project, s.w.Project)

		if di.DeprecationStatus.State == "" && di.DeprecationStatus.ForceSendFields == nil {
			di.DeprecationStatus.ForceSendFields = []string{"Status"}
		}
	}
	return nil
}

func (d *DeprecateImages) validate(ctx context.Context, s *Step) dErr {
	deprecationStates := []string{"", "DEPRECATED", "OBSOLETE", "DELETED"}
	for _, di := range *d {
		if exists, err := projectExists(s.w.ComputeClient, di.Project); err != nil {
			return errf("cannot deprecate image %q: bad project lookup: %q, error: %v", di.Image, di.Project, err)
		} else if !exists {
			return errf("cannot deprecate image %q: project does not exist: %q", di.Image, di.Project)
		}

		if !strIn(di.DeprecationStatus.State, deprecationStates) {
			return errf("DeprecationStatus.State of %q not in %q", di.DeprecationStatus.State, deprecationStates)
		}

		// registerUsage needs the partal url of a non daisy resource.
		lookup := di.Image
		if _, ok := images[s.w].get(di.Image); !ok {
			lookup = fmt.Sprintf("projects/%s/global/images/%s", di.Project, di.Image)
		}
		if _, err := images[s.w].registerUsage(lookup, s); err != nil {
			return newErr(err)
		}
	}

	return nil
}

func (d *DeprecateImages) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)
	for _, di := range *d {
		wg.Add(1)
		go func(di *DeprecateImage) {
			defer wg.Done()

			s.w.Logger.Printf("DeprecateImages: %q --> %q.", di.Image, di.DeprecationStatus.State)
			if err := w.ComputeClient.DeprecateImage(di.Project, di.Image, &di.DeprecationStatus); err != nil {
				e <- newErr(err)
			}
		}(di)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		return nil
	}
}
