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
	"regexp"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// ApplyAndValidateVars is a WorkflowHook that applies vars to a daisy workflow.
// To ensure consistency across worker instances, if vars omits network, subnet, or the
// compute service account, the modifier will automatically apply these values.
type ApplyAndValidateVars struct {
	env  EnvironmentSettings
	vars map[string]string
}

// PreRunHook applies daisy vars to a workflow
func (t *ApplyAndValidateVars) PreRunHook(wf *daisy.Workflow) error {

	// All CLI tools use these variables; if they're declared in the daisy workflow, but not passed by the caller in `vars`,
	// then apply them using the EnvironmentSettings.
	//
	// For network and subnet, some workflows use the prefix `import_`.
	t.backfillVar(regexp.MustCompile("^(import_)?network$"), t.env.Network, wf)
	t.backfillVar(regexp.MustCompile("^(import_)?subnet$"), t.env.Subnet, wf)
	t.backfillVar(regexp.MustCompile("compute_service_account"), t.env.ComputeServiceAccount, wf)
Loop:
	for k, v := range t.vars {
		for wv := range wf.Vars {
			if k == wv {
				wf.AddVar(k, v)
				continue Loop
			}
		}
		return daisy.Errf("unknown workflow Var %q passed to Workflow %q", k, wf.Name)
	}

	return nil
}

func (t *ApplyAndValidateVars) backfillVar(keyPattern *regexp.Regexp, val string, wf *daisy.Workflow) {
	if val == "" {
		return
	}
	for k := range wf.Vars {
		if keyPattern.MatchString(k) && t.vars[k] == "" {
			t.vars[k] = val
			return
		}
	}
}
