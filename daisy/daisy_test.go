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

package main

import (
	"context"
	"reflect"
	"testing"
)

func TestSplitVariables(t *testing.T) {
	var tests = []struct {
		input string
		want  map[string]string
	}{
		{"", map[string]string{}},
		{",", map[string]string{}},
		{"key=var", map[string]string{"key": "var"}},
		{"key1=var1,key2=var2", map[string]string{"key1": "var1", "key2": "var2"}},
	}

	for _, tt := range tests {
		got := splitVariables(tt.input)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("splitVariables did not split %q as expected, want: %q, got: %q", tt.input, tt.want, got)
		}
	}
}

func TestParseWorkflows(t *testing.T) {
	path := "./workflow/test.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	w, err := parseWorkflow(context.Background(), path, varMap, project, zone, gcsPath, oauth, "", "")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		want, got string
	}{
		{w.Project, project},
		{w.Zone, zone},
		{w.GCSPath, gcsPath},
		{w.OAuthPath, oauth},
	}

	for _, tt := range tests {
		if tt.want != tt.got {
			t.Errorf("%s != %v", varMap, w.Vars)
		}
	}

	if reflect.DeepEqual(w.Vars, varMap) {
		t.Errorf("unexpected vars, want: %s, got: %v", varMap, w.Vars)
	}

	want:= "dialing: cannot read service account file: open oauthpath: no such file or directory"
	if _, err := parseWorkflow(context.Background(), path, varMap, project, zone, gcsPath, oauth, "noplace", ""); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}
	
	if _, err := parseWorkflow(context.Background(), path, varMap, project, zone, gcsPath, oauth, "", "noplace"); err.Error() != want {
		t.Errorf("did not get expected error, got: %q, want: %q", err.Error(), want)
	}
}
