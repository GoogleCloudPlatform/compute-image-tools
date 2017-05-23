package workflow

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

type stepImpl interface {
	run(s *Step) error
	validate(s *Step) error
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
	CopyGCSObjects         *CopyGCSObjects         `json:",omitempty"`
	DeleteResources        *DeleteResources        `json:",omitempty"`
	MergeWorkflow          *MergeWorkflow          `json:",omitempty"`
	RunTests               *RunTests               `json:",omitempty"`
	SubWorkflow            *SubWorkflow            `json:",omitempty"`
	WaitForInstancesSignal *WaitForInstancesSignal `json:",omitempty"`
	// Used for unit tests.
	testType stepImpl
}

func (s *Step) stepImpl() (stepImpl, error) {
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
	if s.CopyGCSObjects != nil {
		matchCount++
		result = s.CopyGCSObjects
	}
	if s.DeleteResources != nil {
		matchCount++
		result = s.DeleteResources
	}
	if s.MergeWorkflow != nil {
		matchCount++
		result = s.MergeWorkflow
	}
	if s.RunTests != nil {
		matchCount++
		result = s.RunTests
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
		return nil, errors.New("no step type defined")
	}
	if matchCount > 1 {
		return nil, errors.New("multiple step types defined")
	}
	return result, nil
}

func (s *Step) depends(other *Step) bool {
	deps := s.w.Dependencies
	steps := s.w.Steps
	q := deps[s.name]
	seen := map[string]bool{}

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

func (s *Step) run() error {
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
	s.w.logger.Printf("Running step %q (%s)", s.name, st)
	if err = impl.run(s); err != nil {
		return s.wrapRunError(err)
	}
	select {
	case <-s.w.Cancel:
	default:
		s.w.logger.Printf("Step %q (%s) successfully finished.", s.name, st)
	}
	return nil
}

func (s *Step) validate() error {
	s.w.logger.Printf("Validating step %q", s.name)
	if !rfc1035Rgx.MatchString(strings.ToLower(s.name)) {
		return s.wrapValidateError(errors.New("step name must start with a letter and only contain letters, numbers, and hyphens"))
	}
	impl, err := s.stepImpl()
	if err != nil {
		return s.wrapValidateError(err)
	}
	if err = impl.validate(s); err != nil {
		return s.wrapValidateError(err)
	}
	return nil
}

func (s *Step) wrapRunError(e error) error {
	return fmt.Errorf("step %q run error: %s", s.name, e)
}

func (s *Step) wrapValidateError(e error) error {
	return fmt.Errorf("step %q validation error: %s", s.name, e)
}
