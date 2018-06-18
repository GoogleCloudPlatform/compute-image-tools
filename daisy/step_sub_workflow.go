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
)

// SubWorkflow defines a Daisy sub workflow.
type SubWorkflow struct {
	Path     string
	Vars     map[string]string `json:",omitempty"`
	Workflow *Workflow
}

func (s *SubWorkflow) populate(ctx context.Context, st *Step) dErr {
	if s.Path != "" {
		var err error
		if s.Workflow, err = st.w.NewSubWorkflowFromFile(s.Path); err != nil {
			return newErr(err)
		}
	}

	if s.Workflow == nil {
		return errf("SubWorkflow %q does not have a workflow", st.name)
	}

	s.Workflow.parent = st.w
	s.Workflow.GCSPath = fmt.Sprintf("gs://%s/%s", s.Workflow.parent.bucket, s.Workflow.parent.scratchPath)
	s.Workflow.Name = st.name
	s.Workflow.Project = s.Workflow.parent.Project
	s.Workflow.Zone = s.Workflow.parent.Zone
	s.Workflow.OAuthPath = s.Workflow.parent.OAuthPath
	s.Workflow.ComputeClient = s.Workflow.parent.ComputeClient
	s.Workflow.StorageClient = s.Workflow.parent.StorageClient
	s.Workflow.Logger = s.Workflow.parent.Logger
	s.Workflow.DefaultTimeout = st.Timeout

	var errs dErr
Loop:
	for k, v := range s.Vars {
		for wv := range s.Workflow.Vars {
			if k == wv {
				s.Workflow.AddVar(k, v)
				continue Loop
			}
		}
		errs = addErrs(errs, errf("unknown workflow Var %q passed to SubWorkflow %q", k, st.name))
	}
	if errs != nil {
		return errs
	}

	return s.Workflow.populate(ctx)
}

func (s *SubWorkflow) validate(ctx context.Context, st *Step) dErr {
	return s.Workflow.validate(ctx)
}

func (s *SubWorkflow) run(ctx context.Context, st *Step) dErr {
	// Prerun work has already been done. Just run(), not Run().
	defer s.Workflow.cleanup()
	// If the workflow fails before the subworkflow completes, the previous
	// "defer" cleanup won't happen. Add a failsafe here, have the workflow
	// also call this subworkflow's cleanup.
	st.w.addCleanupHook(func() dErr {
		s.Workflow.cleanup()
		return nil
	})

	st.w.LogStepInfo(st.name, "SubWorkflow", "Running subworkflow %q", s.Workflow.Name)
	if err := s.Workflow.run(ctx); err != nil {
		s.Workflow.LogStepInfo(st.name, "SubWorkflow", "Error running subworkflow %q: %v", s.Workflow.Name, err)
		return err
	}
	return nil
}
