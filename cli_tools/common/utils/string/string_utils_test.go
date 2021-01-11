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

	"github.com/stretchr/testify/assert"
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
			s1:   []string{"MULTI_IP_SUBNET"},
			s2:   []string{"WINDOWS"},
			want: []string{"MULTI_IP_SUBNET", "WINDOWS"},
		},
		{
			s1:   []string{"MULTI_IP_SUBNET", "UEFI_COMPATIBLE"},
			s2:   []string{"WINDOWS", "UEFI_COMPATIBLE"},
			want: []string{"MULTI_IP_SUBNET", "UEFI_COMPATIBLE", "WINDOWS"},
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

func TestSubstring(t *testing.T) {
	assert.Equal(t, "The main string", Substring("The main string", 0, 15))
	assert.Equal(t, "The main string", Substring("The main string", 0, 21))
	assert.Equal(t, "The main string", Substring("The main string", -5, 21))
	assert.Equal(t, "The main str", Substring("The main str", 0, 12))
	assert.Equal(t, " main string", Substring("The main string", 3, 12))
	assert.Equal(t, " main string", Substring("The main string", 3, 50))
	assert.Equal(t, " main s", Substring("The main string", 3, 7))
	assert.Equal(t, "", Substring("The main string", 17, 3))
	assert.Equal(t, "", Substring("The main string", 3, 0))
	assert.Equal(t, "", Substring("The main string", 3, -5))

	assert.Equal(t, "Стефановић", Substring("Вук Стефановић Караџић", 4, 10))
	assert.Equal(t, "什么名", Substring("你叫什么名字", 2, 3))
}

func TestSafeStringToInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"", 0},
		{"0", 0},
		{"10,", 0},
		{"ten,", 0},
		{"10", 10},
		{"-1", -1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SafeStringToInt(tt.input))
		})
	}
}
