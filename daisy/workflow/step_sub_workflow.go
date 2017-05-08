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

// SubWorkflow defines a Daisy sub workflow.
type SubWorkflow struct {
	Path     string
	Vars     map[string]string `json:",omitempty"`
	workflow *Workflow
}

func (s *SubWorkflow) validate(w *Workflow) error {
	return s.workflow.validate()
}

func (s *SubWorkflow) run(w *Workflow) error {
	// Prerun work has already been done. Just run(), not Run().
	defer s.workflow.cleanup()
	w.logger.Printf("Running subworkflow %q", s.workflow.Name)
	if err := s.workflow.run(); err != nil {
		s.workflow.logger.Printf("Error running subworkflow %q: %v", s.workflow.Name, err)
		return err
	}
	return nil
}
