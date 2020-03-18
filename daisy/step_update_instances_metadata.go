//  Copyright 2019 Google Inc. All Rights Reserved.
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

	"google.golang.org/api/compute/v1"
)

// UpdateInstancesMetadata is a Daisy UpdateInstancesMetadata workflow step.
type UpdateInstancesMetadata []*UpdateInstanceMetadata

// UpdateInstanceMetadata is used to update an instance metadata.
type UpdateInstanceMetadata struct {
	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// Instance to attach to.
	Instance      string
	project, zone string
}

func (c *UpdateInstancesMetadata) populate(ctx context.Context, s *Step) DError {
	// We need to validate for duplicates against the existing metadata so we will do this in the running.
	return nil
}

func (c *UpdateInstancesMetadata) validate(ctx context.Context, s *Step) (errs DError) {
	for _, sm := range *c {
		if len(sm.Metadata) == 0 {
			errs = addErrs(errs, Errf("Instance %v: Metadata must contain at least one value to update", sm.Instance))
		}

		ir, err := s.w.instances.regUse(sm.Instance, s)
		if ir == nil {
			// Return now, the rest of this function can't be run without ir.
			return addErrs(errs, Errf("cannot set metadata: %v", err))
		}
		addErrs(errs, err)

		// Set instance project and zone.
		instance := NamedSubexp(instanceURLRgx, ir.link)
		sm.project = instance["project"]
		sm.zone = instance["zone"]
	}
	return errs
}

func (c *UpdateInstancesMetadata) run(ctx context.Context, s *Step) DError {
	var wg sync.WaitGroup
	w := s.w
	e := make(chan DError)
	for _, sm := range *c {
		wg.Add(1)
		go func(sm *UpdateInstanceMetadata) {
			defer wg.Done()

			inst := sm.Instance
			if instRes, ok := w.instances.get(sm.Instance); ok {
				inst = instRes.link
				sm.Instance = instRes.RealName
			}

			// Get metadata fingerprint and original metadata
			resp, err := w.ComputeClient.GetInstance(sm.project, sm.zone, sm.Instance)
			if err != nil {
				e <- newErr("failed to get instance data", err)
				return
			}
			metadata := compute.Metadata{}
			metadata.Fingerprint = resp.Metadata.Fingerprint
			for k, v := range sm.Metadata {
				vCopy := v
				metadata.Items = append(metadata.Items, &compute.MetadataItems{Key: k, Value: &vCopy})
			}

			for _, item := range resp.Metadata.Items {
				// Put only keys that were not updated
				if _, ok := sm.Metadata[item.Key]; !ok {
					metadata.Items = append(metadata.Items, item)
				}
			}

			w.LogStepInfo(s.name, "UpdateInstancesMetadata", "Set Instance %q metadata to %q.", inst, sm.Metadata)
			if err := w.ComputeClient.SetInstanceMetadata(sm.project, sm.zone, sm.Instance, &metadata); err != nil {
				e <- newErr("failed to set instance metadata", err)
				return
			}
		}(sm)
	}

	go func() {
		wg.Wait()
		e <- nil
	}()

	select {
	case err := <-e:
		return err
	case <-w.Cancel:
		wg.Wait()
		return nil
	}
}
