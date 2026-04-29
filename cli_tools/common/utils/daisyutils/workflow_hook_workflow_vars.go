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

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

var (
	// These patterns match a key of "Vars" in a daisy workflow. All CLI tools use these variables.
	//
	// For network and subnet, some workflows use the prefix `import_`.
	networkVarPattern               = regexp.MustCompile("^(import_)?network$")
	subnetVarPattern                = regexp.MustCompile("^(import_)?subnet$")
	computeServiceAccountVarPattern = regexp.MustCompile("compute_service_account")
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
	t.updateNetworkAndSubnet(wf)
	if t.env.ComputeServiceAccount != "" {
		t.backfillVar(computeServiceAccountVarPattern, t.env.ComputeServiceAccount, wf)
	}
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

// updateNetworkAndSubnet updates vars with the network and subnet from the environment.
// It has special logic to explicitly set network to the empty string when the user
// specified the subnet but did not specify the network. This is required to ensure the
// GCE API infers the network from the subnet, rather than trying to use the 'default'
// network from the GCE project.
//
// For more information on this inference, see:
//
//	https://cloud.google.com/compute/docs/reference/rest/v1/instances
func (t *ApplyAndValidateVars) updateNetworkAndSubnet(wf *daisy.Workflow) {
	if t.env.Subnet != "" {
		t.backfillVar(subnetVarPattern, t.env.Subnet, wf)
		if t.env.Network == "" {
			t.backfillVar(networkVarPattern, "", wf)
		}
	}
	if t.env.Network != "" {
		t.backfillVar(networkVarPattern, t.env.Network, wf)
	}
}

// backfillVar searches for a declared daisy variable that matches keyPattern. If a match is found,
// the `vars` is updated with val.
func (t *ApplyAndValidateVars) backfillVar(keyPattern *regexp.Regexp, val string, wf *daisy.Workflow) {
	for k := range wf.Vars {
		if keyPattern.MatchString(k) && t.vars[k] == "" {
			t.vars[k] = val
			return
		}
	}
}
