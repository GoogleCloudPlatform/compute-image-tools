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
	return runMultiTasksStepImpl(c, ctx, s)
}

func (c *CreateDisks) iterateAllTasks(ctx context.Context, f func(context.Context, interface{})) {
	for _, t := range *c {
		f(ctx, t)
	}
}

func (c *CreateDisks) runTask(ctx context.Context, t interface{}, s *Step) DError {
	cd, ok := t.(*Disk)
	if !ok {
		return nil
	}

	// Get the source image link if using a source image.
	if cd.SourceImage != "" {
		if image, ok := s.w.images.get(cd.SourceImage); ok {
			cd.SourceImage = image.link
		}
	}

	s.w.LogStepInfo(s.name, "CreateDisks", "Creating disk %q.", cd.Name)
	if err := s.w.ComputeClient.CreateDisk(cd.Project, cd.Zone, &cd.Disk); err != nil {
		return newErr("failed to create disk", err)
	}

	return nil
}

func (c *CreateDisks) waitAllTasksBeforeCleanup() bool {
	return false
}
