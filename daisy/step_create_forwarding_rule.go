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

// CreateForwardingRules is a Daisy CreateForwardingRules workflow step.
type CreateForwardingRules []*ForwardingRule

func (c *CreateForwardingRules) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, fr := range *c {
		errs = addErrs(errs, fr.populate(ctx, s))
	}
	return errs
}

func (c *CreateForwardingRules) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, fr := range *c {
		errs = addErrs(errs, fr.validate(ctx, s))
	}
	return errs
}

func (c *CreateForwardingRules) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, fr := range *c {
		wg.Add(1)
		go func(fr *ForwardingRule) {
			defer wg.Done()

			w.LogStepInfo(s.name, "CreateForwardingRules", "Creating forwarding-rule %q.", fr.Name)
			if err := w.ComputeClient.CreateForwardingRule(fr.Project, fr.Region, &fr.ForwardingRule); err != nil {
				e <- newErr("failed to create forwarding rules", err)
				return
			}
			fr.createdInWorkflow = true
		}(fr)
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
