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

package daisy

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/api/compute/v1"
)

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

func TestMinInt(t *testing.T) {
	tests := []struct {
		desc string
		x    int
		ys   []int
		want int
	}{
		{"single int case", 1, nil, 1},
		{"first int case", 2, []int{1}, 1},
		{"same ints case", 2, []int{2}, 2},
		{"third int case", 4, []int{3, 2}, 2},
	}

	for _, tt := range tests {
		if got := minInt(tt.x, tt.ys...); got != tt.want {
			t.Errorf("%s: %d != %d", tt.desc, got, tt.want)
		}
	}
}

func TestHasVariableDeclaration(t *testing.T) {
	tests := []struct {
		desc string
		s    string
		want bool
	}{
		{"no declaration", "content", false},
		{"no declaration: empty string", "", false},
		{"no declaration: only dollar", "$var", false},
		{"no declaration: no closing bracket", "{", false},
		{"contains declaration", "content ${k}", true},
		{"contains declaration: source", "${SOURCE: fname}", true},
	}

	for _, tt := range tests {
		if got := hasVariableDeclaration(tt.s); got != tt.want {
			t.Errorf("%s: %v != %v", tt.desc, got, tt.want)
		}
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

func TestStrIn(t *testing.T) {
	ss := []string{"hello", "world", "my", "name", "is", "daisy"}

	// True case.
	if !strIn("hello", ss) {
		t.Fatal("hello not found in slice")
	}

	// False case.
	if strIn("dne", ss) {
		t.Fatal("dne found in slice")
	}

	// Edge case -- empty slice.
	if strIn("dne", []string{}) {
		t.Fatal("string found in empty slice")
	}
}

func TestStrLitPtr(t *testing.T) {
	s1 := "foo"
	s2 := strLitPtr(s1)
	if *s2 != s1 {
		t.Errorf("%q != %q", *s2, s1)
	}
}

func TestStrOr(t *testing.T) {

	tests := []struct {
		desc     string
		s        string
		ss       []string
		expected string
	}{
		{"just one string", "foo", nil, "foo"},
		{"second string is result", "", []string{"bar", "baz"}, "bar"},
		{"third string is result", "", []string{"", "baz"}, "baz"},
		{"all empty", "", []string{""}, ""},
	}

	for _, tt := range tests {
		result := strOr(tt.s, tt.ss...)
		if result != tt.expected {
			t.Errorf("%s: wanted %q, got %q", tt.desc, tt.expected, result)
		}
	}
}

func TestSubstituteSourceVars(t *testing.T) {
	type test struct {
		String string
	}

	tests := []struct {
		got, want test
		wantErr   bool
	}{
		{ // 0
			test{String: "${SOURCE:foo}"},
			test{String: "this is a test"},
			false,
		},
		{ // 1
			test{String: "${BADSOURCE:foo}"},
			test{String: "${BADSOURCE:foo}"},
			false,
		},
		{ // 2
			test{String: "${SOURCE:bar}"},
			test{String: "${SOURCE:bar}"},
			true,
		},
		{ // 3
			test{String: "${SOURCE:baz}"},
			test{String: "${SOURCE:baz}"},
			true,
		},
		{ // 4
			test{String: "${SOURCE:big}"},
			test{String: "${SOURCE:big}"},
			true,
		},
		{ // 5
			test{String: "Did you know that ${SOURCE:foo}?"},
			test{String: "Did you know that this is a test?"},
			false,
		},
		{ // 6
			test{String: "Now with this expansion it crossed the limits: ${SOURCE:almost}?"},
			test{String: "Now with this expansion it crossed the limits: ${SOURCE:almost}?"},
			true,
		},
		{ // 7
			test{String: "${SOURCE:foo} and ${SOURCE:fu}"},
			test{String: "this is a test and this is another test"},
			false,
		},
	}

	td, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(td)
	almost := filepath.Join(td, "almost_too_big")
	ioutil.WriteFile(almost, []byte(strings.Repeat("a", (1024*256)-10)), 0600)

	big := filepath.Join(td, "big_string")
	ioutil.WriteFile(big, []byte(strings.Repeat("a", (1024*256)+10)), 0600)

	ctx := context.Background()
	w := testWorkflow()
	w.Sources = map[string]string{
		"foo":    "./test_data/test.txt",
		"fu":     "./test_data/test_2.txt",
		"bar":    "./test_data/notexist.txt",
		"big":    big,
		"almost": almost}
	for i, tt := range tests {
		s := reflect.ValueOf(&tt.got).Elem()
		err := w.substituteSourceVars(ctx, s)
		if !tt.wantErr && err != nil {
			t.Fatalf("test %d: %v", i, err)
		} else if tt.wantErr && err == nil {
			t.Fatalf("test %d: expected error", i)
		}

		if diffRes := diff(tt.got, tt.want, 0); diffRes != "" {
			t.Errorf("test %d: post substituteSourceVars workflow does not match expectation: (-got +want)\n%s", i, diffRes)
		}
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
		{ // 0
			strings.NewReplacer(),
			test{String: "", private: ""},
			test{String: "", private: ""},
		},
		{ // 1
			strings.NewReplacer(),
			test{String: "string"},
			test{String: "string"},
		},
		{ // 2
			strings.NewReplacer("key", "value"),
			test{String: "key-string", private: "private-key-string"},
			test{String: "value-string", private: "private-key-string"},
		},
		{ // 3
			strings.NewReplacer("key", "value"),
			test{String: "key-key"},
			test{String: "value-value"},
		},
		{ // 4
			strings.NewReplacer("key1", "value1", "key2", "value2"),
			test{String: "key1-key2"},
			test{String: "value1-value2"},
		},
		{ // 5
			strings.NewReplacer(),
			test{StringMap: map[string]string{"key1": "value1"}},
			test{StringMap: map[string]string{"key1": "value1"}},
		},
		{ // 6
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{StringMap: map[string]string{"key1": "key2key2", "key3": "value"}},
			test{StringMap: map[string]string{"value1": "value2value2", "value3": "value"}},
		},
		{ // 7
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{SliceStringMap: map[string][]string{"key": {"value1", "value2"}}},
			test{SliceStringMap: map[string][]string{"key": {"value1", "value2"}}},
		},
		{ // 8
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
		{ // 9
			strings.NewReplacer("key1", "value1", "key2", "value2", "key3", "value3"),
			test{
				StringMap:      map[string]string{"key1": "key2key2", "key3": "value"},
				SliceStringMap: map[string][]string{"key": {"value1", "value2"}},
				Steps: map[string]*Step{
					"create disks key1": {
						CreateDisks: &CreateDisks{
							{
								Disk: compute.Disk{
									Name: "key1",
								},
							},
						},
					},
					"key2": {
						CreateInstances: &CreateInstances{
							Instances: []*Instance{
								{
									Instance: compute.Instance{
										Disks: []*compute.AttachedDisk{{Source: "key1"}},
									},
									Metadata: map[string]string{"test_metadata": "key3"},
								},
							},
						},
					},
					"typed slice step": {
						WaitForInstancesSignal: &WaitForInstancesSignal{
							{Name: "key1"}, {Name: "foo-instance"}, {Name: "key2"},
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
								Disk: compute.Disk{
									Name: "value1",
								},
							},
						},
					},
					"value2": {
						CreateInstances: &CreateInstances{
							Instances: []*Instance{
								{
									Instance: compute.Instance{
										Disks: []*compute.AttachedDisk{{Source: "value1"}},
									},
									Metadata: map[string]string{"test_metadata": "value3"},
								},
							},
						},
					},
					"typed slice step": {
						WaitForInstancesSignal: &WaitForInstancesSignal{
							{Name: "value1"}, {Name: "foo-instance"}, {Name: "value2"},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		s := reflect.ValueOf(&tt.got).Elem()
		substitute(s, tt.replacer)

		if diffRes := diff(tt.got, tt.want, 0); diffRes != "" {
			t.Errorf("test %d: post substitute workflow does not match expectation: (-got +want)\n%s", i, diffRes)
		}
	}
}

func TestCombineGuestOSFeatures(t *testing.T) {

	tests := []struct {
		currentFeatures    []*compute.GuestOsFeature
		additionalFeatures []string
		want               []*compute.GuestOsFeature
	}{
		{
			currentFeatures:    featuresOf(),
			additionalFeatures: []string{},
			want:               featuresOf(),
		},
		{
			currentFeatures:    featuresOf("WINDOWS"),
			additionalFeatures: []string{},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf(),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("WINDOWS"),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("MULTI_IP_SUBNET"),
			additionalFeatures: []string{"WINDOWS"},
			want:               featuresOf("MULTI_IP_SUBNET", "WINDOWS"),
		},
		{
			currentFeatures:    featuresOf("MULTI_IP_SUBNET", "UEFI_COMPATIBLE"),
			additionalFeatures: []string{"WINDOWS", "UEFI_COMPATIBLE"},
			want:               featuresOf("MULTI_IP_SUBNET", "UEFI_COMPATIBLE", "WINDOWS"),
		},
	}

	for _, test := range tests {
		got := CombineGuestOSFeatures(test.currentFeatures, test.additionalFeatures...)

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("CombineGuestOSFeatures(%v, %v) = %v, want %v",
				test.currentFeatures, test.additionalFeatures, got, test.want)
		}
	}
}

func featuresOf(features ...string) []*compute.GuestOsFeature {
	ret := make([]*compute.GuestOsFeature, 0)
	for _, feature := range features {
		ret = append(ret, &compute.GuestOsFeature{
			Type: feature,
		})
	}
	return ret
}
