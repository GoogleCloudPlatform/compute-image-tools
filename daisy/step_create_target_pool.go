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

// CreateTargetPools is a Daisy CreateTargetPools workflow step.
type CreateTargetPools []*TargetPool

func (c *CreateTargetPools) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, tp := range *c {
		errs = addErrs(errs, tp.populate(ctx, s))
	}
	return errs
}

func (c *CreateTargetPools) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, tp := range *c {
		errs = addErrs(errs, tp.validate(ctx, s))
	}
	return errs
}

func (c *CreateTargetPools) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, tp := range *c {
		wg.Add(1)
		go func(tp *TargetPool) {
			defer wg.Done()

			w.LogStepInfo(s.name, "CreateTargetPools", "Creating target pools %q.", tp.Name)
			if err := w.ComputeClient.CreateTargetPool(tp.Project, tp.Region, &tp.TargetPool); err != nil {
				e <- newErr("failed to create target pools", err)
				return
			}
			tp.createdInWorkflow = true
		}(tp)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so target pools being created now can be deleted.
		wg.Wait()
		return nil
	}
}

