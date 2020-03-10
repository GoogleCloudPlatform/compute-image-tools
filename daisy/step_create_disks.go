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
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

const (
	pdStandard = "pd-standard"
	pdSsd      = "pd-ssd"
)

// CreateDisks is a Daisy CreateDisks workflow step via GA API.
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

			// Get the source image link if using a source image.
			if cd.SourceImage != "" {
				if image, ok := w.images.get(cd.SourceImage); ok {
					cd.SourceImage = image.link
				}
			}

			if err := w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
				// Fallback to pd-standard to avoid quota issue.
				if cd.FallbackToPdStandard && strings.HasSuffix(cd.Type, pdSsd) && compute.IsCausedByOperationCode(err, "QUOTA_EXCEEDED") {
					w.LogStepInfo(s.name, "CreateDisks", "Falling back to pd-standard for disk %v. "+
						"It may be caused by insufficient pd-ssd quota. Consider increasing pd-ssd quota to "+
						"avoid using ps-standard for better performance.", cd.Name)
					cd.Type = strings.TrimRight(cd.Type, pdSsd) + pdStandard
					err = w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk)
				}

				if err != nil {
					e <- newErr("failed to create disk", err)
					return
				}
			}
			cd.createdInWorkflow = true
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

// CreateDisksAlpha is a Daisy CreateDisks workflow step via Alpha API.
type CreateDisksAlpha []*DiskAlpha

func (c *CreateDisksAlpha) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, d := range *c {
		errs = addErrs(errs, d.populate(ctx, s))
	}
	return errs
}

func (c *CreateDisksAlpha) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, d := range *c {
		errs = addErrs(errs, d.validate(ctx, s))
	}
	return errs
}

func (c *CreateDisksAlpha) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, d := range *c {
		wg.Add(1)
		go func(cd *DiskAlpha) {
			defer wg.Done()

			// Get the source image link if using a source image.
			if cd.SourceImage != "" {
				if image, ok := w.images.get(cd.SourceImage); ok {
					cd.SourceImage = image.link
				}
			}

			if err := w.ComputeClient.CreateDiskAlpha(cd.Project, cd.Zone, &cd.Disk); err != nil {
				// Fallback to pd-standard to avoid quota issue.
				if cd.FallbackToPdStandard && strings.HasSuffix(cd.Type, pdSsd) && compute.IsCausedByOperationCode(err, "QUOTA_EXCEEDED") {
					w.LogStepInfo(s.name, "CreateDisks", "Falling back to pd-standard for disk %v. "+
						"It may be caused by insufficient pd-ssd quota. Consider increasing pd-ssd quota to "+
						"avoid using ps-standard for better performance.", cd.Name)
					cd.Type = strings.TrimRight(cd.Type, pdSsd) + pdStandard
					err = w.ComputeClient.CreateDiskAlpha(cd.Project, cd.Zone, &cd.Disk)
				}

				if err != nil {
					e <- newErr("failed to create disk", err)
					return
				}
			}
			cd.createdInWorkflow = true
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
