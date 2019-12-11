//  Copyright 2019 Google Inc. All Rights Reserved.
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

package string

import (
	"reflect"
	"testing"
)

func TestCombineStringSlices(t *testing.T) {
	tests := []struct {
		s1   []string
		s2   []string
		want []string
	}{
		{
			s1:   []string{},
			s2:   []string{},
			want: []string{},
		},
		{
			s1:   []string{"WINDOWS"},
			s2:   []string{},
			want: []string{"WINDOWS"},
		},
		{
			s1:   []string{},
			s2:   []string{"WINDOWS"},
			want: []string{"WINDOWS"},
		},
		{
			s1:   []string{"WINDOWS"},
			s2:   []string{"WINDOWS"},
			want: []string{"WINDOWS"},
		},
		{
			s1:   []string{"SECURE_BOOT"},
			s2:   []string{"WINDOWS"},
			want: []string{"SECURE_BOOT", "WINDOWS"},
		},
		{
			s1:   []string{"SECURE_BOOT", "UEFI_COMPATIBLE"},
			s2:   []string{"WINDOWS", "UEFI_COMPATIBLE"},
			want: []string{"SECURE_BOOT", "UEFI_COMPATIBLE", "WINDOWS"},
		},
	}

	for _, test := range tests {
		got := CombineStringSlices(test.s1, test.s2...)

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("CombineStringSlices(%v, %v) = %v, want %v",
				test.s1, test.s2, got, test.want)
		}
	}
}
