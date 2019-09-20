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
	"sort"
	"strconv"
	"sync"

	"google.golang.org/api/compute/v1"
)

// CreateDisks is a Daisy CreateDisks workflow step.
type CreateDisks []*Disk

func (c *CreateDisks) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, d := range *c {
		errs = addErrs(errs, d.populate(ctx, s))
	}
	return errs
}

func (c *CreateDisks) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, d := range *c {
		errs = addErrs(errs, d.validate(ctx, s))
	}
	return errs
}

func (c *CreateDisks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, d := range *c {
		wg.Add(1)
		go func(cd *Disk) {
			defer wg.Done()

			if res, _ := strconv.ParseBool(cd.IsWindows); res {
				cd.GuestOsFeatures = combineGuestOSFeatures(cd.GuestOsFeatures, "WINDOWS")
			}

			// Get the source image link if using a source image.
			if cd.SourceImage != "" {
				if image, ok := w.images.get(cd.SourceImage); ok {
					cd.SourceImage = image.link
				}
			}

			w.LogStepInfo(s.name, "CreateDisks", "Creating disk %q.", cd.Name)
			if err := w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
				e <- newErr("failed to create disk", err)
				return
			}
		}(d)
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

// Merges two slices of features and returns a new slice instance.
// Duplicates are removed, and the result is sorted lexically.
func combineGuestOSFeatures(currentFeatures []*compute.GuestOsFeature,
	additionalFeatures ...string) []*compute.GuestOsFeature {
	featureSet := map[string]bool{}
	for _, feature := range additionalFeatures {
		featureSet[feature] = true
	}
	for _, feature := range currentFeatures {
		featureSet[feature.Type] = true
	}
	ret := make([]*compute.GuestOsFeature, 0)
	for feature := range featureSet {
		ret = append(ret, &compute.GuestOsFeature{
			Type: feature,
		})
	}
	// Sort elements by type, lexically. This ensures
	// stability of output ordering for tests.
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Type < ret[j].Type
	})
	return ret
}
