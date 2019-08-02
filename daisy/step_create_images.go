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
	"sync"

	"google.golang.org/api/googleapi"
)

var UseBetaAPI = false

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages struct {
	Images     []*Image
	ImagesBeta []*ImageBeta
}

// UnmarshalJSON unmarshals Image.
func (ci *CreateImages) UnmarshalJSON(b []byte) error {
	imagesBeta := &[]*ImageBeta{}
	if err := json.Unmarshal(b, imagesBeta); err != nil {
		return err
	}

	if usesBetaFeatures(imagesBeta) {
		// Actually uses Beta features, no need to parse as GA
		ci.ImagesBeta = *imagesBeta
		return nil
	}

	images := &[]*Image{}
	if err := json.Unmarshal(b, images); err == nil {
		ci.Images = *images
		return nil
	} else {
		return err
	}
}

func usesBetaFeatures(imagesBeta *[]*ImageBeta) bool {
	if UseBetaAPI {
		return UseBetaAPI
	}
	for _, imageBeta := range *imagesBeta {
		if imageBeta != nil && len(imageBeta.StorageLocations) > 0 {
			return true
		}
	}
	return false
}

// populate preprocesses fields: Name, Project, Description, SourceDisk, RawDisk, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateImages) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	if c.Images != nil {
		for _, i := range c.Images {
			errs = addErrs(errs, Populate(i, &i.ImageBase, ctx, s))
		}
	}

	if c.ImagesBeta != nil {
		for _, i := range c.ImagesBeta {
			errs = addErrs(errs, Populate(i, &i.ImageBase, ctx, s))
		}
	}

	return errs
}

func (c *CreateImages) validate(ctx context.Context, s *Step) dErr {
	var errs dErr

	if c.Images != nil {
		for _, i := range c.Images {
			errs = addErrs(errs, Validate(i, &i.ImageBase, i.Licenses, ctx, s))
		}
	}

	if c.ImagesBeta != nil {
		for _, i := range c.ImagesBeta {
			errs = addErrs(errs, Validate(i, &i.ImageBase, i.Licenses, ctx, s))
		}
	}

	return errs
}

func (c *CreateImages) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)

	createImage := func(ci ImageInterface, overwrite bool) {
		defer wg.Done()
		// Get source disk link if SourceDisk is a daisy reference to a disk.
		if d, ok := w.disks.get(ci.getSourceDisk()); ok {
			ci.setSourceDisk(d.link)
		}

		// Delete existing if OverWrite is true.
		if overwrite {
			// Just try to delete it, a 404 here indicates the image doesn't exist.
			if err := ci.delete(w.ComputeClient); err != nil {
				if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
					e <- errf("error deleting existing image: %v", err)
					return
				}
			}
		}

		w.LogStepInfo(s.name, "CreateImages", "Creating image %q.", ci.getName())
		if err := ci.create(w.ComputeClient); err != nil {
			e <- newErr(err)
			return
		}
	}

	for _, i := range c.Images {
		wg.Add(1)
		go createImage(i, i.OverWrite)
	}

	for _, i := range c.ImagesBeta {
		wg.Add(1)
		go createImage(i, i.OverWrite)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so Images being created now will complete before we try to clean them up.
		wg.Wait()
		return nil
	}
}
