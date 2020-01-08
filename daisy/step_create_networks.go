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
)

// CreateNetworks is a Daisy CreateNetwork workflow step.
type CreateNetworks []*Network

func (c *CreateNetworks) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, n := range *c {
		errs = addErrs(errs, n.populate(ctx, s))
	}
	return errs
}

func (c *CreateNetworks) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, n := range *c {
		errs = addErrs(errs, n.validate(ctx, s))
	}
	return errs
}

func (c *CreateNetworks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, n := range *c {
		wg.Add(1)
		go func(n *Network) {
			defer wg.Done()

			w.LogStepInfo(s.name, "CreateNetworks", "Creating network %q.", n.Name)
			if err := w.ComputeClient.CreateNetwork(n.Project, &n.Network); err != nil {
				e <- newErr("failed to create networks", err)
				return
			}
			n.createdInWorkflow = true
		}(n)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so networks being created now can be deleted.
		wg.Wait()
		return nil
	}
}
