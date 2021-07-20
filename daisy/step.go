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
	"fmt"
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
	populate(ctx context.Context, s *Step) DError
	validate(ctx context.Context, s *Step) DError
	run(ctx context.Context, s *Step) DError
}

// Step is a single daisy workflow step.
type Step struct {
	name string
	w    *Workflow

	//Timeout description
	TimeoutDescription string `json:",omitempty"`
	// Time to wait for this step to complete (default 10m).
	// Must be parsable by https://golang.org/pkg/time/#ParseDuration.
	Timeout string `json:",omitempty"`
	timeout time.Duration
	// Only one of the below fields should exist for each instance of Step.
	AttachDisks               *AttachDisks               `json:",omitempty"`
	DetachDisks               *DetachDisks               `json:",omitempty"`
	CreateDisks               *CreateDisks               `json:",omitempty"`
	CreateForwardingRules     *CreateForwardingRules     `json:",omitempty"`
	CreateFirewallRules       *CreateFirewallRules       `json:",omitempty"`
	CreateHealthChecks        *CreateHealthChecks        `json:",omitempty"`
	CreateImages              *CreateImages              `json:",omitempty"`
	CreateMachineImages       *CreateMachineImages       `json:",omitempty"`
	CreateInstances           *CreateInstances           `json:",omitempty"`
	CreateNetworks            *CreateNetworks            `json:",omitempty"`
	CreateSnapshots           *CreateSnapshots           `json:",omitempty"`
	CreateSubnetworks         *CreateSubnetworks         `json:",omitempty"`
	CreateTargetInstances     *CreateTargetInstances     `json:",omitempty"`
	CopyGCSObjects            *CopyGCSObjects            `json:",omitempty"`
	ResizeDisks               *ResizeDisks               `json:",omitempty"`
	StartInstances            *StartInstances            `json:",omitempty"`
	StopInstances             *StopInstances             `json:",omitempty"`
	DeleteResources           *DeleteResources           `json:",omitempty"`
	DeprecateImages           *DeprecateImages           `json:",omitempty"`
	IncludeWorkflow           *IncludeWorkflow           `json:",omitempty"`
	SubWorkflow               *SubWorkflow               `json:",omitempty"`
	WaitForInstancesSignal    *WaitForInstancesSignal    `json:",omitempty"`
	WaitForAnyInstancesSignal *WaitForAnyInstancesSignal `json:",omitempty"`
	UpdateInstancesMetadata   *UpdateInstancesMetadata   `json:",omitempty"`
	// Used for unit tests.
	testType stepImpl
}

// NewStep creates a Step with given name and timeout with the specified workflow.
// If timeout is less or equal to zero, defaultTimeout from the workflow will be used
func NewStep(name string, w *Workflow, timeout time.Duration) *Step {
	if timeout <= 0 {
		return &Step{name: name, w: w, Timeout: w.DefaultTimeout}
	}
	return &Step{name: name, w: w, timeout: timeout}
}

// NewStepDefaultTimeout creates a Step with given name using default timeout from the workflow
func NewStepDefaultTimeout(name string, w *Workflow) *Step {
	return NewStep(name, w, 0)
}

func (s *Step) stepImpl() (stepImpl, DError) {
	var result stepImpl
	matchCount := 0
	if s.AttachDisks != nil {
		matchCount++
		result = s.AttachDisks
	}
	if s.DetachDisks != nil {
		matchCount++
		result = s.DetachDisks
	}
	if s.CreateDisks != nil {
		matchCount++
		result = s.CreateDisks
	}
	if s.CreateForwardingRules != nil {
		matchCount++
		result = s.CreateForwardingRules
	}
	if s.CreateFirewallRules != nil {
		matchCount++
		result = s.CreateFirewallRules
	}
	if s.CreateHealthChecks != nil {
		matchCount++
		result = s.CreateHealthChecks
	}
	if s.CreateImages != nil {
		matchCount++
		result = s.CreateImages
	}
	if s.CreateMachineImages != nil {
		matchCount++
		result = s.CreateMachineImages
	}
	if s.CreateInstances != nil {
		matchCount++
		result = s.CreateInstances
	}
	if s.CreateNetworks != nil {
		matchCount++
		result = s.CreateNetworks
	}
	if s.CreateSnapshots != nil {
		matchCount++
		result = s.CreateSnapshots
	}
	if s.CreateSubnetworks != nil {
		matchCount++
		result = s.CreateSubnetworks
	}
	if s.CreateTargetInstances != nil {
		matchCount++
		result = s.CreateTargetInstances
	}
	if s.CopyGCSObjects != nil {
		matchCount++
		result = s.CopyGCSObjects
	}
	if s.ResizeDisks != nil {
		matchCount++
		result = s.ResizeDisks
	}
	if s.StartInstances != nil {
		matchCount++
		result = s.StartInstances
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
	if s.WaitForAnyInstancesSignal != nil {
		matchCount++
		result = s.WaitForAnyInstancesSignal
	}
	if s.UpdateInstancesMetadata != nil {
		matchCount++
		result = s.UpdateInstancesMetadata
	}
	if s.testType != nil {
		matchCount++
		result = s.testType
	}

	if matchCount == 0 {
		return nil, Errf("no step type defined")
	}
	if matchCount > 1 {
		return nil, Errf("multiple step types defined")
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

func (s *Step) populate(ctx context.Context) DError {
	s.w.LogWorkflowInfo("Populating step %q", s.name)
	impl, err := s.stepImpl()
	if err != nil {
		return s.wrapPopulateError(err)
	}
	if err = impl.populate(ctx, s); err != nil {
		err = s.wrapPopulateError(err)
	}
	return err
}

func (s *Step) recordStepTime(startTime time.Time) {
	endTime := time.Now()
	s.w.recordStepTime(s.name, startTime, endTime)
}

func (s *Step) run(ctx context.Context) DError {
	startTime := time.Now()
	defer s.recordStepTime(startTime)
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
	s.w.LogWorkflowInfo("Running step %q (%s)", s.name, st)
	if err = impl.run(ctx, s); err != nil {
		return s.wrapRunError(err)
	}
	select {
	case <-s.w.Cancel:
		// return an error to indicate a canceled workflow is not 'success'
		return s.w.onStepCancel(s, st)
	default:
		s.w.LogWorkflowInfo("Step %q (%s) successfully finished.", s.name, st)
	}
	return nil
}

func (s *Step) validate(ctx context.Context) DError {
	s.w.LogWorkflowInfo("Validating step %q", s.name)
	if !rfc1035Rgx.MatchString(strings.ToLower(s.name)) {
		return s.wrapValidateError(Errf("step name must start with a letter and only contain letters, numbers, and hyphens"))
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

func (s *Step) wrapPopulateError(e DError) DError {
	return wrapErrf(e, "step %q populate error", s.name)
}

func (s *Step) wrapRunError(e DError) DError {
	return wrapErrf(e, "step %q run error", s.name)
}

func (s *Step) wrapValidateError(e DError) DError {
	return wrapErrf(e, "step %q validation error", s.name)
}

func (s *Step) getTimeoutError() DError {
	var timeoutDescription string
	if s.TimeoutDescription != "" {
		timeoutDescription = fmt.Sprintf(". %s", s.TimeoutDescription)
	}

	return Errf("step %q did not complete within the specified timeout of %s%s", s.name, s.timeout, timeoutDescription)
}
