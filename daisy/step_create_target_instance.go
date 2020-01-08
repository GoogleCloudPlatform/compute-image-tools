//  Copyright 2018 Google Inc. All Rights Reserved.
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
)

// CreateTargetInstances is a Daisy CreateTargetInstances workflow step.
type CreateTargetInstances []*TargetInstance

func (c *CreateTargetInstances) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, ti := range *c {
		errs = addErrs(errs, ti.populate(ctx, s))
	}
	return errs
}

func (c *CreateTargetInstances) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, ti := range *c {
		errs = addErrs(errs, ti.validate(ctx, s))
	}
	return errs
}

func (c *CreateTargetInstances) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, ti := range *c {
		wg.Add(1)
		go func(ti *TargetInstance) {
			defer wg.Done()

			w.LogStepInfo(s.name, "CreateTargetInstances", "Creating target instance %q.", ti.Name)
			if err := w.ComputeClient.CreateTargetInstance(ti.Project, ti.Zone, &ti.TargetInstance); err != nil {
				e <- newErr("failed to create target instances", err)
				return
			}
			ti.createdInWorkflow = true
		}(ti)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so target instances being created now can be deleted.
		wg.Wait()
		return nil
	}
}
