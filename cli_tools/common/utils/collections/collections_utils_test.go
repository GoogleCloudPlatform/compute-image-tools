//  Copyright 2020 Google Inc. All Rights Reserved.
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

package collections

import (
	"reflect"
	"sort"
	"testing"
)

func TestReverseMap(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]string
		want          map[string]string
		expectSuccess bool
	}{
		{"nil map", nil, map[string]string{}, true},
		{"empty map", map[string]string{}, map[string]string{}, true},
		{"single item map", map[string]string{"k1": "v1"}, map[string]string{"v1": "k1"}, true},
		{"multiple items map", map[string]string{"k1": "v1", "k2": "v2"}, map[string]string{"v1": "k1", "v2": "k2"}, true},
		{"dup values", map[string]string{"k1": "v1", "k2": "v1"}, nil, false},
	}

	for _, test := range tests {
		m, ok := ReverseMap(test.input)
		if test.expectSuccess != ok {
			t.Errorf("[%v] Expected success: %v, actual: %v", test.name, test.expectSuccess, ok)
		} else if test.expectSuccess && !reflect.DeepEqual(m, test.want) {
			t.Errorf("[%v] Expected map '%v' != actual map '%v'", test.name, test.want, m)
		}
	}
}

func TestGetKeys(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  []string
	}{
		{"nil map", nil, []string{}},
		{"empty map", map[string]string{}, []string{}},
		{"single item map", map[string]string{"k1": "v1"}, []string{"k1"}},
		{"multiple items map", map[string]string{"k1": "v1", "k2": "v2"}, []string{"k1", "k2"}},
	}

	for _, test := range tests {
		keys := GetKeys(test.input)
		sort.Strings(keys)
		sort.Strings(test.want)
		if !reflect.DeepEqual(keys, test.want) {
			t.Errorf("[%v] Expected keys '%v' != actual keys '%v'", test.name, test.want, keys)
		}
	}
}
