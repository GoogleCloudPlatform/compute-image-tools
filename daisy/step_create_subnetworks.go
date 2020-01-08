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

// CreateSubnetworks is a Daisy CreateSubnetwork workflow step.
type CreateSubnetworks []*Subnetwork

func (c *CreateSubnetworks) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, sn := range *c {
		errs = addErrs(errs, sn.populate(ctx, s))
	}
	return errs
}

func (c *CreateSubnetworks) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, sn := range *c {
		errs = addErrs(errs, sn.validate(ctx, s))
	}
	return errs
}

func (c *CreateSubnetworks) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, sn := range *c {
		wg.Add(1)
		go func(sn *Subnetwork) {
			defer wg.Done()

			if networkRes, ok := w.networks.get(sn.Network); ok {
				sn.Network = networkRes.link
			}

			w.LogStepInfo(s.name, "CreateSubnetworks", "Creating subnetwork %q.", sn.Name)
			if err := w.ComputeClient.CreateSubnetwork(sn.Project, sn.Region, &sn.Subnetwork); err != nil {
				e <- newErr("failed to create subnetworks", err)
				return
			}
			sn.createdInWorkflow = true
		}(sn)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so subnetworks being created now can be deleted.
		wg.Wait()
		return nil
	}
}
