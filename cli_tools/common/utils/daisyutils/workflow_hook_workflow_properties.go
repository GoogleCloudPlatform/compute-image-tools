//  Copyright 2021 Google Inc. All Rights Reserved.
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

package daisyutils

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// ApplyEnvToWorkflow is a WorkflowHook that applies user-customizable values
// to the top-level parent workflow.
type ApplyEnvToWorkflow struct {
	env EnvironmentSettings
}

// PreRunHook updates properties on wf that correspond to user-specified values
// such as project, zone, and scratch bucket path.
func (t *ApplyEnvToWorkflow) PreRunHook(wf *daisy.Workflow) error {
	set(t.env.Project, &wf.Project)
	set(t.env.Zone, &wf.Zone)
	set(t.env.GCSPath, &wf.GCSPath)
	set(t.env.OAuth, &wf.OAuthPath)
	set(t.env.Timeout, &wf.DefaultTimeout)
	set(t.env.ComputeEndpoint, &wf.ComputeEndpoint)
	return nil
}

func set(src string, dst *string) {
	if src != "" {
		*dst = src
	}
}
