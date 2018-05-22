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
	"reflect"
	"strings"
	"time"
)

type stepImpl interface {
	// populate modifies the step type field values.
	// populate should set defaults, extend GCE partial URLs to full partial
	// URLs (partial URLs including the "projects/<project>" prefix), etc.
	// This should not perform value validation.
	// Returns any parsing errors.
	populate(ctx context.Context, s *Step) dErr
	validate(ctx context.Context, s *Step) dErr
	run(ctx context.Context, s *Step) dErr
}

// Step is a single daisy workflow step.
type Step struct {
	name string
	w    *Workflow

	// Time to wait for this step to complete (default 10m).
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	Timeout string
	timeout time.Duration
	// Only one of the below fields should exist for each instance of Step.
	AttachDisks            *AttachDisks            `json:",omitempty"`
	CreateDisks            *CreateDisks            `json:",omitempty"`
	CreateImages           *CreateImages           `json:",omitempty"`
	CreateInstances        *CreateInstances        `json:",omitempty"`
	CreateNetworks         *CreateNetworks         `json:",omitempty"`
	CopyGCSObjects         *CopyGCSObjects         `json:",omitempty"`
	StopInstances          *StopInstances          `json:",omitempty"`
	DeleteResources        *DeleteResources        `json:",omitempty"`
	DeprecateImages        *DeprecateImages        `json:",omitempty"`
	IncludeWorkflow        *IncludeWorkflow        `json:",omitempty"`
	SubWorkflow            *SubWorkflow            `json:",omitempty"`
	WaitForInstancesSignal *WaitForInstancesSignal `json:",omitempty"`
	// Used for unit tests.
	testType stepImpl
}

func (s *Step) stepImpl() (stepImpl, dErr) {
	var result stepImpl
	matchCount := 0
	if s.AttachDisks != nil {
		matchCount++
		result = s.AttachDisks
	}
	if s.CreateDisks != nil {
		matchCount++
		result = s.CreateDisks
	}
	if s.CreateImages != nil {
		matchCount++
		result = s.CreateImages
	}
	if s.CreateInstances != nil {
		matchCount++
		result = s.CreateInstances
	}
	if s.CreateNetworks != nil {
		matchCount++
		result = s.CreateNetworks
	}
	if s.CopyGCSObjects != nil {
		matchCount++
		result = s.CopyGCSObjects
	}
	if s.StopInstances != nil {
		matchCount++
		result = s.StopInstances
	}
	if s.DeleteResources != nil {
		matchCount++
		result = s.DeleteResources
	}
	if s.DeprecateImages != nil {
		matchCount++
		result = s.DeprecateImages
	}
	if s.IncludeWorkflow != nil {
		matchCount++
		result = s.IncludeWorkflow
	}
	if s.SubWorkflow != nil {
		matchCount++
		result = s.SubWorkflow
	}
	if s.WaitForInstancesSignal != nil {
		matchCount++
		result = s.WaitForInstancesSignal
	}
	if s.testType != nil {
		matchCount++
		result = s.testType
	}

	if matchCount == 0 {
		return nil, errf("no step type defined")
	}
	if matchCount > 1 {
		return nil, errf("multiple step types defined")
	}
	return result, nil
}

func (s *Step) depends(other *Step) bool {
	if s == nil || other == nil || s.w == nil || s.w != other.w {
		return false
	}
	deps := s.w.Dependencies
	steps := s.w.Steps
	q := deps[s.name]
	seen := map[string]bool{}

	// Do a BFS search on s's dependencies, looking for the target dependency. Don't revisit visited dependencies.
	for i := 0; i < len(q); i++ {
		name := q[i]
		if seen[name] {
			continue
		}
		seen[name] = true
		if steps[name] == other {
			return true
		}
		for _, dep := range deps[name] {
			q = append(q, dep)
		}
	}

	return false
}

// nestedDepends determines if s depends on other, taking into account the recursive, nested nature of
// workflows, i.e. workflows in IncludeWorkflow and SubWorkflow.
// Example: if s depends on an IncludeWorkflow whose workflow contains other, then s depends on other.
func (s *Step) nestedDepends(other *Step) bool {
	sChain := s.getChain()
	oChain := other.getChain()
	// If sChain and oChain don't share the same root workflow, then there is no dependency relationship.
	if len(sChain) == 0 || len(oChain) == 0 || sChain[0].w != oChain[0].w {
		return false
	}

	// Find where the step chains diverge.
	// A divergence in the chains indicates sibling steps, where we can check dependency.
	// We want to see if s's branch depends on other's branch.
	var sStep, oStep *Step
	for i := 0; i < minInt(len(sChain), len(oChain)); i++ {
		sStep = sChain[i]
		oStep = oChain[i]
		if sStep != oStep {
			break
		}
	}
	return sStep.depends(oStep)
}

// getChain returns the step chain getting to a step. A link in the chain represents an IncludeWorkflow step, a
// SubWorkflow step, or the step itself.
// For example, workflow A has a step s1 which includes workflow B. B has a step s2 which subworkflows C. Finally,
// C has a step s3. s3.getChain() will return []*Step{s1, s2, s3}
func (s *Step) getChain() []*Step {
	if s == nil || s.w == nil {
		return nil
	}
	if s.w.parent == nil {
		return []*Step{s}
	}
	for _, st := range s.w.parent.Steps {
		if st.IncludeWorkflow != nil && st.IncludeWorkflow.Workflow == s.w {
			return append(st.getChain(), s)
		}
		if st.SubWorkflow != nil && st.SubWorkflow.Workflow == s.w {
			return append(st.getChain(), s)
		}
	}
	// We shouldn't get here.
	return nil
}

func (s *Step) populate(ctx context.Context) dErr {
	s.w.Logger.WorkflowInfo(s.w, "Populating step %q", s.name)
	impl, err := s.stepImpl()
	if err != nil {
		return s.wrapPopulateError(err)
	}
	if err = impl.populate(ctx, s); err != nil {
		err = s.wrapPopulateError(err)
	}
	return err
}

func (s *Step) run(ctx context.Context) dErr {
	impl, err := s.stepImpl()
	if err != nil {
		return s.wrapRunError(err)
	}
	var st string
	if t := reflect.TypeOf(impl); t.Kind() == reflect.Ptr {
		st = t.Elem().Name()
	} else {
		st = t.Name()
	}
	s.w.Logger.WorkflowInfo(s.w, "Running step %q (%s)", s.name, st)
	if err = impl.run(ctx, s); err != nil {
		return s.wrapRunError(err)
	}
	select {
	case <-s.w.Cancel:
	default:
		s.w.Logger.WorkflowInfo(s.w, "Step %q (%s) successfully finished.", s.name, st)
	}
	return nil
}

func (s *Step) validate(ctx context.Context) dErr {
	s.w.Logger.WorkflowInfo(s.w, "Validating step %q", s.name)
	if !rfc1035Rgx.MatchString(strings.ToLower(s.name)) {
		return s.wrapValidateError(errf("step name must start with a letter and only contain letters, numbers, and hyphens"))
	}
	impl, err := s.stepImpl()
	if err != nil {
		return s.wrapValidateError(err)
	}
	if err = impl.validate(ctx, s); err != nil {
		return s.wrapValidateError(err)
	}
	return nil
}

func (s *Step) wrapPopulateError(e dErr) dErr {
	return errf("step %q populate error: %s", s.name, e)
}

func (s *Step) wrapRunError(e dErr) dErr {
	return errf("step %q run error: %s", s.name, e)
}

func (s *Step) wrapValidateError(e dErr) dErr {
	return errf("step %q validation error: %s", s.name, e)
}
