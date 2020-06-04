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

package assert

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGreaterThanOrEqualTo(t *testing.T) {
	tests := []struct {
		value       int
		limit       int
		expectPanic bool
	}{
		{
			value: 0,
			limit: 0,
		},
		{
			value: 100,
			limit: 0,
		},
		{
			value:       0,
			limit:       1,
			expectPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d >= %d", tt.value, tt.limit), func(t *testing.T) {
			if tt.expectPanic {
				assert.PanicsWithValue(t, fmt.Sprintf("Expected %d >= %d", tt.value, tt.limit), func() {
					GreaterThanOrEqualTo(tt.value, tt.limit)
				})
			} else {
				GreaterThanOrEqualTo(tt.value, tt.limit)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		element     string
		arr         []string
		expectPanic bool
	}{
		{
			element: "",
			arr:     []string{""},
		},
		{
			element:     "",
			arr:         []string{},
			expectPanic: true,
		},
		{
			element:     "item",
			arr:         []string{},
			expectPanic: true,
		},
		{
			element: "item",
			arr:     []string{"item"},
		},
		{
			element: "item",
			arr:     []string{"one", "item"},
		},
		{
			element: "item",
			arr:     []string{"item", "one"},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v contains %s", tt.arr, tt.element), func(t *testing.T) {
			if tt.expectPanic {
				assert.PanicsWithValue(t, fmt.Sprintf("%s is not a member of %v", tt.element, tt.arr), func() {
					Contains(tt.element, tt.arr)
				})
			} else {
				Contains(tt.element, tt.arr)
			}
		})
	}
}
