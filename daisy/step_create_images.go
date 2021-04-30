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

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages struct {
	Images      []*Image
	ImagesAlpha []*ImageAlpha
	ImagesBeta  []*ImageBeta
}

// UnmarshalJSON unmarshals Image.
func (ci *CreateImages) UnmarshalJSON(b []byte) error {
	var imagesAlpha []*ImageAlpha
	if err := json.Unmarshal(b, &imagesAlpha); err != nil {
		return err
	}
	ci.ImagesAlpha = imagesAlpha

	var imagesBeta []*ImageBeta
	if err := json.Unmarshal(b, &imagesBeta); err != nil {
		return err
	}
	ci.ImagesBeta = imagesBeta

	var images []*Image
	if err := json.Unmarshal(b, &images); err != nil {
		return err
	}
	ci.Images = images

	return nil
}

func imageUsesAlphaFeatures(imagesAlpha []*ImageAlpha) bool {
	for _, imageAlpha := range imagesAlpha {
		if imageAlpha != nil && imageAlpha.RolloutOverride != nil && len(imageAlpha.RolloutOverride.DefaultRolloutTime) > 0 {
			return true
		}
		if imageAlpha != nil && imageAlpha.Deprecated != nil && imageAlpha.Deprecated.StateOverride != nil && len(imageAlpha.Deprecated.StateOverride.DefaultRolloutTime) > 0 {
			return true
		}
	}
	return false
}

func imageUsesBetaFeatures(imagesBeta []*ImageBeta) bool {
	return false
}

// populate preprocesses fields: Name, Project, Description, SourceDisk, RawDisk, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (ci *CreateImages) populate(ctx context.Context, s *Step) DError {
	var errs DError
	if ci.Images != nil {
		for _, i := range ci.Images {
			errs = addErrs(errs, (&i.ImageBase).populate(ctx, i, s))
		}
	}

	if ci.ImagesAlpha != nil {
		for _, i := range ci.ImagesAlpha {
			errs = addErrs(errs, (&i.ImageBase).populate(ctx, i, s))
		}
	}

	if ci.ImagesBeta != nil {
		for _, i := range ci.ImagesBeta {
			errs = addErrs(errs, (&i.ImageBase).populate(ctx, i, s))
		}
	}

	return errs
}

func (ci *CreateImages) validate(ctx context.Context, s *Step) DError {
	var errs DError

	if imageUsesBetaFeatures(ci.ImagesBeta) {
		for _, i := range ci.ImagesBeta {
			errs = addErrs(errs, (&i.ImageBase).validate(ctx, i, i.Licenses, s))
		}
	} else {
		for _, i := range ci.Images {
			errs = addErrs(errs, (&i.ImageBase).validate(ctx, i, i.Licenses, s))
		}
	}

	return errs
}

func (ci *CreateImages) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)

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
					e <- Errf("error deleting existing image: %v", err)
					return
				}
			}
		}

		w.LogStepInfo(s.name, "CreateImages", "Creating image %q.", ci.getName())
		if err := ci.create(w.ComputeClient); err != nil {
			e <- newErr("failed to create images", err)
			return
		}
		ci.markCreatedInWorkflow()
	}

	if imageUsesAlphaFeatures(ci.ImagesAlpha) {
		for _, i := range ci.ImagesAlpha {
			wg.Add(1)
			go createImage(i, i.OverWrite)
		}
	} else if imageUsesBetaFeatures(ci.ImagesBeta) {
		for _, i := range ci.ImagesBeta {
			wg.Add(1)
			go createImage(i, i.OverWrite)
		}
	} else {
		for _, i := range ci.Images {
			wg.Add(1)
			go createImage(i, i.OverWrite)
		}
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
