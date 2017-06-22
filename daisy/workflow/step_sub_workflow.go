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

package workflow

import "fmt"

// SubWorkflow defines a Daisy sub workflow.
type SubWorkflow struct {
	Path string
	Vars map[string]string `json:",omitempty"`
	w    *Workflow
}

func (s *SubWorkflow) populate(st *Step) error {
	s.w.GCSPath = fmt.Sprintf("gs://%s/%s", s.w.parent.bucket, s.w.parent.scratchPath)
	s.w.Name = st.name
	s.w.Project = s.w.parent.Project
	s.w.Zone = s.w.parent.Zone
	s.w.OAuthPath = s.w.parent.OAuthPath
	s.w.ComputeClient = s.w.parent.ComputeClient
	s.w.StorageClient = s.w.parent.StorageClient
	s.w.gcsLogWriter = s.w.parent.gcsLogWriter
	s.w.vars = map[string]vars{}
	for k, v := range s.Vars {
		s.w.vars[k] = vars{Value: v}
	}
	return s.w.populate()
}

func (s *SubWorkflow) validate(st *Step) error {
	return s.w.validate()
}

func (s *SubWorkflow) run(st *Step) error {
	// Prerun work has already been done. Just run(), not Run().
	defer s.w.cleanup()
	// If the workflow fails before the subworkflow completes, the previous
	// "defer" cleanup won't happen. Add a failsafe here, have the workflow
	// also call this subworkflow's cleanup.
	st.w.addCleanupHook(func() error {
		s.w.cleanup()
		return nil
	})

	st.w.logger.Printf("Running subworkflow %q", s.w.Name)
	if err := s.w.run(); err != nil {
		s.w.logger.Printf("Error running subworkflow %q: %v", s.w.Name, err)
		close(st.w.Cancel)
		return err
	}
	return nil
}
