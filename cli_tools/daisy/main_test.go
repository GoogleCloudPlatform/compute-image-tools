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
	"flag"
	"fmt"
	"reflect"
	"testing"
)

func TestPopulateVars(t *testing.T) {
	var tests = []struct {
		input string
		want  map[string]string
	}{
		{"", map[string]string{"test1": "value"}},
		{",", map[string]string{"test1": "value"}},
		{"key=var", map[string]string{"test1": "value", "key": "var"}},
		{"key1=var1,key2=var2", map[string]string{"test1": "value", "key1": "var1", "key2": "var2"}},
	}

	// Add a generated var flag.
	flag.String("var:test1", "", "")
	flag.CommandLine.Parse([]string{"-var:test1", "value"})

	for _, tt := range tests {
		got := populateVars(tt.input)
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("splitVariables did not split %q as expected, want: %q, got: %q", tt.input, tt.want, got)
		}
	}
}

func TestAddFlags(t *testing.T) {
	firstFlag := "var:first_var"
	secondFlag := "var:second_var"
	value := "value"

	flag.Bool("var:test2", false, "")
	flag.CommandLine.Parse([]string{"-validate", "-var:test2"})

	args := []string{"-validate", "-var:test2", "-" + firstFlag, value, fmt.Sprintf("--%s=%s", secondFlag, value), "var:not_a_flag", "also_not_a_flag"}
	before := flag.NFlag()
	addFlags(args)
	flag.CommandLine.Parse(args)
	after := flag.NFlag()

	want := before + 2
	if after != want {
		t.Errorf("number of flags after does not match expectation, want %d, got %d", want, after)
	}

	for _, fn := range []string{firstFlag, secondFlag} {
		got := flag.Lookup(fn).Value.String()
		if got != value {
			t.Errorf("flag %q value %q!=%q", fn, got, value)
		}
	}
}

func TestParseWorkflows(t *testing.T) {
	path := "../../daisy/test_data/test.wf.json"
	varMap := map[string]string{"key1": "var1", "key2": "var2"}
	project := "project"
	zone := "zone"
	gcsPath := "gcspath"
	oauth := "oauthpath"
	dTimeout := "10m"
	endpoint := "endpoint"
	w, err := parseWorkflow(context.Background(), path, varMap, project, zone, gcsPath, oauth, dTimeout, endpoint, true, true, true)
	if err != nil {
		t.Fatal(err)
	}

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
