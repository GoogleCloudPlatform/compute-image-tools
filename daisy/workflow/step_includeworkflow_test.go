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

import "testing"

func TestIncludeWorkflowRun(t *testing.T) {}

func TestIncludeWorkflowValidate(t *testing.T) {
	w := testWorkflow()
	disks[w].add("foo", &resource{})
	iw := w.NewIncludedWorkflow()
	w.Steps = map[string]*Step{
		"included": {
			IncludeWorkflow: &IncludeWorkflow{
				w: iw,
			},
		},
	}
	iw.Steps = map[string]*Step{
		"del": {
			DeleteResources: &DeleteResources{
				Disks: []string{"foo"},
			},
		},
	}

	w.populate()
	s := w.Steps["included"]
	if err := s.IncludeWorkflow.populate(s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
