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

// CreateFirewallRules is a Daisy CreateFirewallRules workflow step.
type CreateFirewallRules []*FirewallRule

func (c *CreateFirewallRules) populate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, fir := range *c {
		errs = addErrs(errs, fir.populate(ctx, s))
	}
	return errs
}

func (c *CreateFirewallRules) validate(ctx context.Context, s *Step) DError {
	var errs DError
	for _, fir := range *c {
		errs = addErrs(errs, fir.validate(ctx, s))
	}
	return errs
}

func (c *CreateFirewallRules) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, fir := range *c {
		wg.Add(1)
		go func(fir *FirewallRule) {
			defer wg.Done()

			if networkRes, ok := w.networks.get(fir.Network); ok {
				fir.Network = networkRes.link
			}

			w.LogStepInfo(s.name, "CreateFirewallRules", "Creating firewall rule %q.", fir.Name)
			if err := w.ComputeClient.CreateFirewallRule(fir.Project, &fir.Firewall); err != nil {
				e <- newErr("failed to create firewall", err)
				return
			}
			fir.createdInWorkflow = true
		}(fir)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		// Wait so firewall rules being created now can be deleted.
		wg.Wait()
		return nil
	}
}
