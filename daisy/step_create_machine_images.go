//  Copyright 2020 Google Inc. All Rights Reserved.
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

// CreateMachineImages is a Daisy workflow step for creating machine images.
type CreateMachineImages []*MachineImage

// populate pre-processed fields: Name, Project, Description and daisyName.
// - sets defaults
// - extends short partial URLs to include "projects/<project>"
func (c *CreateMachineImages) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, mi := range *c {
		errs = addErrs(errs, mi.populate(ctx, s))
	}
	return errs
}

func (c *CreateMachineImages) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, i := range *c {
		errs = addErrs(errs, i.validate(ctx, s))
	}
	return errs
}

func (c *CreateMachineImages) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	eChan := make(chan DError)
	for _, ci := range *c {
		wg.Add(1)
		go func(mi *MachineImage) {
			defer wg.Done()

			// Get source instance link if SourceInstance is a Daisy reference to an instance.
			if i, ok := w.instances.get(mi.SourceInstance); ok {
				mi.SourceInstance = i.link
			}

			// Delete existing machine image if OverWrite is true.
			if mi.OverWrite {
				// Just try to delete it, a 404 here indicates the machine image doesn't exist.
				if err := w.ComputeClient.DeleteMachineImage(mi.Project, mi.Name); err != nil {
					if apiErr, ok := err.(*googleapi.Error); !ok || apiErr.Code != 404 {
						eChan <- Errf("error deleting existing machine image: %v", err)
						return
					}
				}
			}

			w.LogStepInfo(s.name, "CreateMachineImages", "Creating machine image %q.", mi.Name)

			if err := w.ComputeClient.CreateMachineImage(mi.Project, &mi.MachineImage); err != nil {
				eChan <- newErr("failed to create machine image", err)
				return
			}
			mi.createdInWorkflow = true
		}(ci)
	}

	go func() {
		wg.Wait()
		eChan <- nil
	}()

	select {
	case err := <-eChan:
		return err
	case <-w.Cancel:
		// Wait so machine images being created now will complete before we try to clean them up.
		wg.Wait()
		return nil
	}
}
