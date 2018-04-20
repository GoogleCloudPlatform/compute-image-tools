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
	"sync"

	"google.golang.org/api/googleapi"
)

// CreateImages is a Daisy CreateImages workflow step.
type CreateImages []*Image

// populate preprocesses fields: Name, Project, Description, SourceDisk, RawDisk, and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateImages) populate(ctx context.Context, s *Step) dErr {
	var errs dErr
	for _, i := range *c {
		errs = addErrs(errs, i.populate(ctx, s))
	}
	return errs
}

func (c *CreateImages) validate(ctx context.Context, s *Step) dErr {
	var errs dErr
	for _, i := range *c {
		errs = addErrs(errs, i.validate(ctx, s))
	}
	return errs
}

func (c *CreateImages) run(ctx context.Context, s *Step) dErr {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan dErr)
	for _, i := range *c {
		wg.Add(1)
		go func(ci *Image) {
			defer wg.Done()
			// Get source disk link if SourceDisk is a daisy reference to a disk.
			if d, ok := w.disks.get(ci.SourceDisk); ok {
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

			w.Logger.StepInfo(s.w, "CreateImages", "creating image %q.", ci.Name)
			if err := w.ComputeClient.CreateImage(ci.Project, &ci.Image); err != nil {
				e <- newErr(err)
				return
			}
		}(i)
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
