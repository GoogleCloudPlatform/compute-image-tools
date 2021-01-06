//  Copyright 2018 Google Inc. All Rights Reserved.
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

package daisycommon

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_ParseWorkflow_HappyCase(t *testing.T) {
	path := "../../daisy/test_data/test.wf.json"
	varMap := map[string]string{"bootstrap_instance_name": "bootstrap-${NAME}", "key1": "var1", "key2": "var2", "machine_type": "n1-standard-1"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow(path, varMap, project, zone, gcsPath, oauth, dTimeout, endpoint, true,
		true, true)
	if err != nil {
		t.Fatal(err)
	}

	assertWorkflow(t, w, project, zone, gcsPath, oauth, dTimeout, endpoint, varMap)
}

func Test_ParseWorkflow_RaisesErrorWhenInvalidPath(t *testing.T) {
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow("/file/not/found", varMap, project, zone, gcsPath, oauth, dTimeout, endpoint,
		true, true, true)
	assert.Nil(t, w)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "/file/not/found: no such file or directory")
}

func Test_ApplyWorkerCustomizations(t *testing.T) {
	for _, tt := range []struct {
		name                       string
		env                        EnvironmentSettings
		originalVars, expectedVars map[string]string
	}{
		{
			name: "create daisy.Var when either network or subnetwork is empty",
			env: EnvironmentSettings{
				Network:               "",
				Subnet:                "",
				ComputeServiceAccount: "csa",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{
				"network":                 "",
				"subnet":                  "",
				"compute_service_account": "csa",
			},
		},
		{
			name: "overwrite daisy.Var when either network or subnetwork is empty",
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
				"network":                 "",
				"subnet":                  "",
				"compute_service_account": "csa",
			},
		},
		{
			name: "don't create daisy.Var when compute_service_account is empty",
			env: EnvironmentSettings{
				Network:               "net",
				Subnet:                "sub",
				ComputeServiceAccount: "",
			},
			originalVars: map[string]string{},
			expectedVars: map[string]string{
				"network": "net",
				"subnet":  "sub",
			},
		},
		{
			name: "don't overwrite daisy.Var when compute_service_account is empty",
			env: EnvironmentSettings{
				Network:               "net",
				Subnet:                "sub",
				ComputeServiceAccount: "",
			},
			originalVars: map[string]string{
				"compute_service_account": "default",
			},
			expectedVars: map[string]string{
				"network":                 "net",
				"subnet":                  "sub",
				"compute_service_account": "default",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := daisy.Workflow{}
			for k, v := range tt.originalVars {
				wf.AddVar(k, v)
			}
			tt.env.ApplyWorkerCustomizations(&wf)
			expectedVars := tt.expectedVars
			assertEqualWorkflowVars(t, &wf, expectedVars)
		})
	}
}

func Test_ApplyToWorkflow(t *testing.T) {
	for _, tt := range []struct {
		name               string
		env                EnvironmentSettings
		original, expected *daisy.Workflow
	}{
		{
			name: "always overwrite when source fields are non-empty",
			env: EnvironmentSettings{
				Project:         "lucky-lemur",
				Zone:            "us-west1-c",
				GCSPath:         "new-path",
				OAuth:           "new-oauth",
				Timeout:         "new-timeout",
				ComputeEndpoint: "new-endpoint",
			},
			original: &daisy.Workflow{
				Project:         "original-project",
				Zone:            "original-zone",
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
			expected: &daisy.Workflow{
				Project:         "lucky-lemur",
				Zone:            "us-west1-c",
				GCSPath:         "new-path",
				OAuthPath:       "new-oauth",
				DefaultTimeout:  "new-timeout",
				ComputeEndpoint: "new-endpoint",
			},
		},
		{
			name: "project and zone overwrite when empty",
			env:  EnvironmentSettings{},
			original: &daisy.Workflow{
				Project:         "original-project",
				Zone:            "original-zone",
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
			expected: &daisy.Workflow{
				GCSPath:         "original-path",
				OAuthPath:       "original-oauth",
				DefaultTimeout:  "original-timeout",
				ComputeEndpoint: "original-endpoint",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.env.ApplyToWorkflow(tt.original)
			assert.Equal(t, tt.original, tt.expected)
		})
	}
}

func Test_ApplyToWorkflow_PropagatesLogging(t *testing.T) {
	original := &daisy.Workflow{}

	// ApplyToWorkflow calls methods to disable logging, which in turn updates private
	// fields on daisy.Workflow. This test inspects private fields directly
	// to validate that logging is disabled.
	privateLoggingFields := []string{"gcsLoggingDisabled", "stdoutLoggingDisabled", "cloudLoggingDisabled"}
	for _, fieldName := range privateLoggingFields {
		realValue := reflect.ValueOf(original).Elem().FieldByName(fieldName)
		assert.False(t, realValue.Bool(), "field: %s", fieldName)
	}

	EnvironmentSettings{
		DisableGCSLogs:    true,
		DisableCloudLogs:  true,
		DisableStdoutLogs: true,
	}.ApplyToWorkflow(original)

	for _, fieldName := range privateLoggingFields {
		realValue := reflect.ValueOf(original).Elem().FieldByName(fieldName)
		assert.True(t, realValue.Bool(), "field: %s", fieldName)
	}
}

func assertWorkflow(t *testing.T, w *daisy.Workflow, project string, zone string, gcsPath string,
	oauth string, dTimeout string, endpoint string, varMap map[string]string) {
	tests := []struct {
		want, got interface{}
	}{
		{w.Project, project},
		{w.Zone, zone},
		{w.GCSPath, gcsPath},
		{w.OAuthPath, oauth},
		{w.DefaultTimeout, dTimeout},
		{w.ComputeEndpoint, endpoint},
	}
	for _, tt := range tests {
		if tt.want != tt.got {
			t.Errorf("%v != %v", tt.want, tt.got)
		}
	}
	assertEqualWorkflowVars(t, w, varMap)
}

func assertEqualWorkflowVars(t *testing.T, wf *daisy.Workflow, expectedVars map[string]string) {
	actualVars := map[string]string{}
	for k, v := range wf.Vars {
		actualVars[k] = v.Value
	}
	assert.Equal(t, actualVars, expectedVars)
}
