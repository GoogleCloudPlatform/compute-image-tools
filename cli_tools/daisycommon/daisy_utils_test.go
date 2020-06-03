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

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

func TestParseWorkflows(t *testing.T) {
	path := "../../daisy/test_data/test.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
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

func TestParseWorkflowsInvalidPath(t *testing.T) {
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := ParseWorkflow("NOT_VALID_PATH", varMap, project, zone, gcsPath, oauth, dTimeout, endpoint,
		true, true, true)
	assert.Nil(t, w)
	assert.NotNil(t, err)
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
	if reflect.DeepEqual(w.Vars, varMap) {
		t.Errorf("unexpected vars, want: %v, got: %v", varMap, w.Vars)
	}
}
