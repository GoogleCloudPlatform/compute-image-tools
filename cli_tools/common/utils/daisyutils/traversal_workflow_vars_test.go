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
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_ApplyEnvToWorkers_SetsNetworkAndAccounts(t *testing.T) {
	for _, tt := range []struct {
		name                       string
		env                        EnvironmentSettings
		declaredDaisyVars          []string
		originalVars, expectedVars map[string]string
	}{
		{
			name:              "backfill env variables when declared in workflow",
			declaredDaisyVars: []string{"network", "subnet", "compute_service_account"},
			env: EnvironmentSettings{
				Network:               "a",
				Subnet:                "b",
				ComputeServiceAccount: "c",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{
				"network":                 "a",
				"subnet":                  "b",
				"compute_service_account": "c",
			},
		},
		{
			name:              "don't overwrite existing variables passed to traversal",
			declaredDaisyVars: []string{"network", "subnet", "compute_service_account"},
			env: EnvironmentSettings{
				Network:               "a",
				Subnet:                "b",
				ComputeServiceAccount: "c",
			},
			originalVars: map[string]string{
				"network":                 "x",
				"subnet":                  "y",
				"compute_service_account": "z",
			},
			expectedVars: map[string]string{
				"network":                 "x",
				"subnet":                  "y",
				"compute_service_account": "z",
			},
		},
		{
			name: "ignore env variables if not declared in workflow",
			env: EnvironmentSettings{
				Network:               "a",
				Subnet:                "b",
				ComputeServiceAccount: "c",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{},
		},
		{
			name:              "overwrite daisy.Var when either network or subnetwork is empty",
			declaredDaisyVars: []string{"network", "subnet", "compute_service_account"},
			env: EnvironmentSettings{
				Network:               "",
				Subnet:                "",
				ComputeServiceAccount: "csa",
			},
			originalVars: map[string]string{
				"network": "default",
				"subnet":  "regional",
			},
			expectedVars: map[string]string{
				"network":                 "default",
				"subnet":                  "regional",
				"compute_service_account": "csa",
			},
		},
		{
			name:              "support `import_network` and `import_subnet` naming",
			declaredDaisyVars: []string{"import_network", "import_subnet"},
			env: EnvironmentSettings{
				Network: "a",
				Subnet:  "b",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{
				"import_network": "a",
				"import_subnet":  "b",
			},
		},
		{
			name:              "don't update variables that have network or subnet as a substring",
			declaredDaisyVars: []string{"a_network_var", "a_subnet_var"},
			env: EnvironmentSettings{
				Network: "a",
				Subnet:  "b",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{
				"a_network_var": "",
				"a_subnet_var":  "",
			},
		},
		{
			name:              "apply non-env variables to workfly",
			declaredDaisyVars: []string{"var1", "var2"},
			env: EnvironmentSettings{
				Network: "a",
				Subnet:  "b",
			},
			originalVars: map[string]string{
				"var1": "a",
				"var2": "b",
			},
			expectedVars: map[string]string{
				"var1": "a",
				"var2": "b",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := &daisy.Workflow{}
			for _, varname := range tt.declaredDaisyVars {
				wf.AddVar(varname, "")
			}
			(&ApplyAndValidateVars{tt.env, tt.originalVars}).Traverse(wf)
			assertEqualWorkflowVars(t, wf, tt.expectedVars)
		})
	}
}
