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

	computeAlpha "google.golang.org/api/compute/v0.alpha"
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
	// DeprecationStatus to set for image.
	DeprecationStatusAlpha computeAlpha.DeprecationStatus
	// Project image is in, overrides workflow Project.
	Project string `json:",omitempty"`
}

func (d *DeprecateImages) populate(ctx context.Context, s *Step) DError {
	for _, di := range *d {
		di.Project = strOr(di.Project, s.w.Project)
	}
	return nil
}

func (d *DeprecateImages) validate(ctx context.Context, s *Step) DError {
	deprecationStates := []string{"", "DEPRECATED", "OBSOLETE", "DELETED"}
	for _, di := range *d {
		if exists, err := projectExists(s.w.ComputeClient, di.Project); err != nil {
			return Errf("cannot deprecate image %q: bad project lookup: %q, error: %v", di.Image, di.Project, err)
		} else if !exists {
			return Errf("cannot deprecate image %q: project does not exist: %q", di.Image, di.Project)
		}

		// Verify State is one of the deprecated states.
		// The Alpha check also requires the value to not be emptry string as in that case the GA API will be used.
		if di.DeprecationStatusAlpha.State != "" && strIn(di.DeprecationStatusAlpha.State, deprecationStates) {
				return Errf("DeprecationStatusAlpha.State of %q not in %q", di.DeprecationStatusAlpha.State, deprecationStates)
		} else if !strIn(di.DeprecationStatus.State, deprecationStates) {
				return Errf("DeprecationStatus.State of %q not in %q", di.DeprecationStatus.State, deprecationStates)
		}

		// regUse needs the partal url of a non daisy resource.
		lookup := di.Image
		if _, ok := s.w.images.get(di.Image); !ok {
			lookup = fmt.Sprintf("projects/%s/global/images/%s", di.Project, di.Image)
		}
		if _, err := s.w.images.regUse(lookup, s); err != nil {
			return newErr("failed to register use of image when deprecating", err)
		}
	}

	return nil
}

func (d *DeprecateImages) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, di := range *d {
		wg.Add(1)
		go func(di *DeprecateImage) {
			defer wg.Done()
			if di.DeprecationStatusAlpha.State != "" {
				w.LogStepInfo(s.name, "DeprecateImages", "%q --> %q with DefaultRolloutTime %s.", di.Image, di.DeprecationStatusAlpha.State, di.DeprecationStatusAlpha.StateOverride.DefaultRolloutTime)
				if err := w.ComputeClient.DeprecateImageAlpha(di.Project, di.Image, &di.DeprecationStatusAlpha); err != nil {
					e <- newErr("failed to deprecate images", err)
				}
			} else {
				w.LogStepInfo(s.name, "DeprecateImages", "%q --> %q.", di.Image, di.DeprecationStatus.State)
				if err := w.ComputeClient.DeprecateImage(di.Project, di.Image, &di.DeprecationStatus); err != nil {
					e <- newErr("failed to deprecate images", err)
				}
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
