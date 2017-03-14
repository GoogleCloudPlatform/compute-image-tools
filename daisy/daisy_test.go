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
	"reflect"
	"testing"
)

func TestSplitVariables(t *testing.T) {
	var tests = []struct {
		input string
		want  map[string]string
	}{
		{"", map[string]string{}},
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
