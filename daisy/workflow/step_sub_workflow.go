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
	Workflow *Workflow `json:"-"`
}

func (s *SubWorkflow) validate() error {
	return s.Workflow.validate()
}

func (s *SubWorkflow) run(w *Workflow) error {
	// As this is a sub workflow, we need to copy all resources from the main
	// workflow at start, and back at the end.
	copy(s.Workflow.createdInstances, w.createdInstances)
	s.Workflow.createdDisks = map[string]string{}
	for name, link := range w.createdDisks {
		s.Workflow.createdDisks[name] = link
	}
	s.Workflow.createdImages = map[string]string{}
	for name, link := range w.createdImages {
		s.Workflow.createdImages[name] = link
	}

	defer func() {
		for _, name := range s.Workflow.createdInstances {
			w.addCreatedInstance(name)
		}
		for name, link := range s.Workflow.createdDisks {
			w.addCreatedDisk(name, link)
		}
		for name, link := range s.Workflow.createdImages {
			w.addCreatedImage(name, link)
		}
	}()

	return s.Workflow.traverseDAG(func(st step) error { return s.Workflow.runStep(st.(*Step)) })
}
