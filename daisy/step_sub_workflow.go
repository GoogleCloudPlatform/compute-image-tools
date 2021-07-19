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
	Workflow *Workflow         `json:",omitempty"`
}

func (s *SubWorkflow) populate(ctx context.Context, st *Step) DError {
	if s.Path != "" && s.Workflow == nil {
		var err error
		if s.Workflow, err = st.w.NewSubWorkflowFromFile(s.Path); err != nil {
			return ToDError(err)
		}
	}

	if s.Workflow == nil {
		return Errf("SubWorkflow %q does not have a workflow", st.name)
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

	var errs DError
Loop:
	for k, v := range s.Vars {
		for wv := range s.Workflow.Vars {
			if k == wv {
				s.Workflow.AddVar(k, v)
				continue Loop
			}
		}
		errs = addErrs(errs, Errf("unknown workflow Var %q passed to SubWorkflow %q", k, st.name))
	}
	if errs != nil {
		return errs
	}

	return s.Workflow.populate(ctx)
}

func (s *SubWorkflow) validate(ctx context.Context, st *Step) DError {
	return s.Workflow.validate(ctx)
}

func (s *SubWorkflow) run(ctx context.Context, st *Step) DError {
	if err := s.Workflow.uploadSources(ctx); err != nil {
		return err
	}

	swCleanup := func() {
		s.Workflow.LogWorkflowInfo("SubWorkflow %q cleaning up (this may take up to 2 minutes).", s.Workflow.Name)
		for _, hook := range s.Workflow.cleanupHooks {
			if err := hook(); err != nil {
				s.Workflow.LogWorkflowInfo("Error returned from SubWorkflow cleanup hook: %s", err)
			}
		}
	}

	defer swCleanup()
	// If the workflow fails before the subworkflow completes, the previous
	// "defer" cleanup won't happen. Add a failsafe here, have the workflow
	// also call this subworkflow's cleanup.
	st.w.addCleanupHook(func() DError {
		swCleanup()
		return nil
	})

	// Prerun work has already been done. Just run(), not Run().
	st.w.LogStepInfo(st.name, "SubWorkflow", "Running subworkflow %q", s.Workflow.Name)
	if err := s.Workflow.run(ctx); err != nil {
		s.Workflow.LogStepInfo(st.name, "SubWorkflow", "Error running subworkflow %q: %v", s.Workflow.Name, err)
		return err
	}
	return nil
}
