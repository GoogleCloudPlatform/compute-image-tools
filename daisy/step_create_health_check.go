//  Copyright 2021 Google Inc. All Rights Reserved.
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

// CreateHealthChecks is a Daisy CreateHealthChecks workflow step.
type CreateHealthChecks []*HealthCheck

func (c *CreateHealthChecks) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, hc := range *c {
		errs = addErrs(errs, hc.populate(ctx, s))
	}
	return errs
}

func (c *CreateHealthChecks) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, hc := range *c {
		errs = addErrs(errs, hc.validate(ctx, s))
	}
	return errs
}

func (c *CreateHealthChecks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, hc := range *c {
		wg.Add(1)
		go func(hc *HealthCheck) {
			defer wg.Done()

			w.LogStepInfo(s.name, "CreateHealthChecks", "Creating health-check %q.", hc.Name)
			if err := w.ComputeClient.CreateHealthCheck(hc.Project, &hc.HealthCheck); err != nil {
				e <- newErr("failed to create health checks", err)
				return
			}
			hc.createdInWorkflow = true
		}(hc)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so forwarding-rules being created now can be deleted.
		wg.Wait()
		return nil
	}
}
