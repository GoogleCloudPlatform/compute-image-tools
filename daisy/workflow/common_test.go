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

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestContainsString(t *testing.T) {
	ss := []string{"hello", "world", "my", "name", "is", "daisy"}

	// True case.
	if !containsString("hello", ss) {
		t.Fatal("hello not found in slice")
	}

	// False case.
	if containsString("dne", ss) {
		t.Fatal("dne found in slice")
	}

	// Edge case -- empty slice.
	if containsString("dne", []string{}) {
		t.Fatal("string found in empty slice")
	}
}

func TestRandString(t *testing.T) {
	for i := 0; i < 10; i++ {
		l := len(randString(i))
		if l != i {
			t.Fatalf("wrong string length: %d != %d", l, i)
		}
	}
}

func TestFilter(t *testing.T) {
	ss := []string{"my", "name", "is", "daisy", "what", "is", "yours"}

	tests := []struct {
		desc, toFilter string
		want           []string
	}{
		{"normal case", "daisy", []string{"my", "name", "is", "what", "is", "yours"}},
		{"edge -- start of slice", "my", []string{"name", "is", "daisy", "what", "is", "yours"}},
		{"edge -- end of slice", "yours", []string{"my", "name", "is", "daisy", "what", "is"}},
		{"filter multiple", "is", []string{"my", "name", "daisy", "what", "yours"}},
		{"filter string that DNE", "dne", []string{"my", "name", "is", "daisy", "what", "is", "yours"}},
	}

	for _, tt := range tests {
		result := filter(ss, tt.toFilter)
		if !reflect.DeepEqual(result, tt.want) {
			t.Errorf("%s failed. input: s=%s; got: %s; want: %s", tt.desc, tt.toFilter, result, tt.want)
		}
	}

	// Edge case -- empty slice.
	result := filter([]string{}, "hello")
	if !reflect.DeepEqual(result, []string{}) {
		t.Error("remove on empty slice failed")
	}
}

func TestSubstitute(t *testing.T) {
	type test struct {
		String         string
		StringMap      map[string]string
		SliceStringMap map[string][]string
		Steps          map[string]*Step
		private        string
	}

	tests := []struct {
		replacer  *strings.Replacer
		got, want test
	}{
		{ // 1
			strings.NewReplacer(),
			test{String: "", private: ""},
			test{String: "", private: ""},
		},
		{ // 2
			strings.NewReplacer(),
			test{String: "string"},
			test{String: "string"},
		},
		{ // 3
			strings.NewReplacer("key", "value"),
			test{String: "key-string", private: "private-key-string"},
			test{String: "value-string", private: "private-key-string"},
		},
		{ // 4
			strings.NewReplacer("key", "value"),
			test{String: "key-key"},
			test{String: "value-value"},
		},
		{ // 5
			strings.NewReplacer("key1", "value1", "key2", "value2"),
			test{String: "key1-key2"},
			test{String: "value1-value2"},
		},
		{ // 6
			strings.NewReplacer(),
			test{StringMap: map[string]string{"key1": "value1"}},
			test{StringMap: map[string]string{"key1": "value1"}},
		},
		{ // 7
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{StringMap: map[string]string{"key1": "key2key2", "key3": "value"}},
			test{StringMap: map[string]string{"value1": "value2value2", "value3": "value"}},
		},
		{ // 8
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{SliceStringMap: map[string][]string{"key": {"value1", "value2"}}},
			test{SliceStringMap: map[string][]string{"key": {"value1", "value2"}}},
		},
		{ // 9
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{
				SliceStringMap: map[string][]string{
					"key1": {"key1", "value2"},
					"key2": {"key2", "value2"},
				},
			},
			test{
				SliceStringMap: map[string][]string{
					"value1": {"value1", "value2"},
					"value2": {"value2", "value2"},
				},
			},
		},
		{ // 10
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{
				StringMap:      map[string]string{"key1": "key2key2", "key3": "value"},
				SliceStringMap: map[string][]string{"key": {"value1", "value2"}},
				Steps: map[string]*Step{
					"create disks key1": {
						CreateDisks: &CreateDisks{
							{
								Name: "key1",
							},
						},
					},
					"key2": {
						CreateInstances: &CreateInstances{
							{
								AttachedDisks: []string{"key1"},
								Metadata:      map[string]string{"test_metadata": "key3"},
							},
						},
					},
					"step3": {
						SubWorkflow: &SubWorkflow{
							Path: "key3",
							workflow: &Workflow{
								Name: "key1",
								Steps: map[string]*Step{
									"key1": {
										Timeout: "key3",
										CreateImages: &CreateImages{
											{
												Name: "key1",
											},
										},
									},
								},
							},
						},
					},
					"typed slice step": {
						WaitForInstancesSignal: &WaitForInstancesSignal{
							"key1", "foo-instance", "key2",
						},
					},
				},
			},
			test{
				StringMap:      map[string]string{"value1": "value2value2", "value3": "value"},
				SliceStringMap: map[string][]string{"key": {"value1", "value2"}},
				Steps: map[string]*Step{
					"create disks value1": {
						CreateDisks: &CreateDisks{
							{
								Name: "value1",
							},
						},
					},
					"value2": {
						CreateInstances: &CreateInstances{
							{
								AttachedDisks: []string{"value1"},
								Metadata:      map[string]string{"test_metadata": "value3"},
							},
						},
					},
					"step3": {
						SubWorkflow: &SubWorkflow{
							Path: "value3",
							workflow: &Workflow{
								Name: "key1", // substitution should not recurse into subworkflows
								Steps: map[string]*Step{
									"key1": { // substitution should not recurse into subworkflows
										Timeout: "key3", // substitution should not recurse into subworkflows
										CreateImages: &CreateImages{
											{
												Name: "key1", // substitution should not recurse into subworkflows
											},
										},
									},
								},
							},
						},
					},
					"typed slice step": {
						WaitForInstancesSignal: &WaitForInstancesSignal{
							"value1", "foo-instance", "value2",
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		s := reflect.ValueOf(&tt.got).Elem()
		substitute(s, tt.replacer)

		if diff := pretty.Compare(tt.got, tt.want); diff != "" {
			t.Errorf("test %d: post substitute workflow does not match expectation: (-got +want)\n%s", i+1, diff)
		}
	}
}

func TestSplitGCSPath(t *testing.T) {
	tests := []struct {
		input     string
		bucket    string
		object    string
		shouldErr bool
	}{
		{"gs://foo", "foo", "", false},
		{"gs://foo/bar", "foo", "bar", false},
		{"http://foo.storage.googleapis.com/bar", "foo", "bar", false},
		{"https://foo.storage.googleapis.com/bar", "foo", "bar", false},
		{"http://storage.googleapis.com/foo/bar", "foo", "bar", false},
		{"https://storage.googleapis.com/foo/bar", "foo", "bar", false},
		{"http://commondatastorage.googleapis.com/foo/bar", "foo", "bar", false},
		{"https://commondatastorage.googleapis.com/foo/bar", "foo", "bar", false},
		{"/local/path", "", "", true},
	}

	for _, tt := range tests {
		b, o, err := splitGCSPath(tt.input)
		if tt.shouldErr && err == nil {
			t.Errorf("splitGCSPath(%q) should have thrown an error", tt.input)
		} else if !tt.shouldErr && err != nil {
			t.Errorf("splitGCSPath(%q) should not have thrown an error", tt.input)
		}
		if b != tt.bucket || o != tt.object {
			t.Errorf("splitGCSPath(%q) returned incorrect values -- want bucket=%q, object=%q; got bucket=%q, object=%q", tt.input, tt.bucket, tt.object, b, o)
		}
	}
}
