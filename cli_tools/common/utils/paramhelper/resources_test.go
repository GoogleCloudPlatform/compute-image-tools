//  Copyright 2021 Google Inc. All Rights Reserved.
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

package paramhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimResourcePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		}, {
			input:    "https://compute.googleapis.com/compute/v1/",
			expected: "",
		}, {
			input:    "https://compute.googleapis.com/compute/v1/rest",
			expected: "rest",
		}, {
			input:    "https://www.googleapis.com/compute/v1/",
			expected: "",
		}, {
			input:    "https://www.googleapis.com/compute/v1/rest",
			expected: "rest",
		}, {
			input:    "//compute.googleapis.com/compute/",
			expected: "",
		}, {
			input:    "//compute.googleapis.com/compute/rest",
			expected: "rest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, trimResourcePrefix(tt.input))
		})
	}
}
