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
	"os"
	"path"
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

func TestDirectoryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	tmpFileObj, err := os.CreateTemp("", "*.txt")
	assert.NoError(t, err)
	tmpFile := tmpFileObj.Name()
	tests := []struct {
		name        string
		dir         string
		expectPanic string
	}{
		{
			name: "Don't panic when directory exists",
			dir:  tmpDir,
		},
		{
			name:        "Panic when dir doesn't exist",
			dir:         path.Join(tmpDir, "dir-doesn't-exist"),
			expectPanic: fmt.Sprintf("%s/dir-doesn't-exist: Directory not found", tmpDir),
		},
		{
			name:        "Panic when dir is a file",
			dir:         tmpFile,
			expectPanic: fmt.Sprintf("%v: Directory not found", tmpFile),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic != "" {
				assert.PanicsWithValue(t, tt.expectPanic, func() {
					DirectoryExists(tt.dir)
				})
			} else {
				DirectoryExists(tt.dir)
			}
		})
	}
}

func TestNotEmpty(t *testing.T) {
	tests := []struct {
		name        string
		obj         interface{}
		expectPanic bool
	}{
		{
			obj:  "s",
			name: "pass when non-empty string",
		},
		{
			obj:  struct{ s string }{s: "hi"},
			name: "pass when non-default struct",
		},
		{
			obj:  map[string]string{"hi": "there"},
			name: "pass when non-empty map",
		},
		{
			obj:  [1]string{"hi"},
			name: "pass when non-empty array",
		}, {
			obj:  []string{"hi"},
			name: "pass when non-empty slice",
		},
		{
			obj:         nil,
			name:        "panic when object is untypped nil",
			expectPanic: true,
		},
		{
			obj:         "",
			name:        "pass when empty string",
			expectPanic: true,
		},
		{
			obj:         map[string]string{},
			name:        "panic when object is empty map",
			expectPanic: true,
		},
		{
			obj:         map[string]map[string]string{},
			name:        "panic when object is empty nested map",
			expectPanic: true,
		},
		{
			obj:         [0]string{},
			name:        "panic when object is empty array",
			expectPanic: true,
		}, {
			obj:         []string{},
			name:        "panic when object is empty slice",
			expectPanic: true,
		},
		{
			obj:         [][]string{},
			name:        "panic when object is empty nested slice",
			expectPanic: true,
		},
		{
			obj:         struct{ s string }{},
			name:        "panic when default struct",
			expectPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.PanicsWithValue(t, "Expected non-empty value", func() {
					NotEmpty(tt.obj)
				})
			} else {
				NotEmpty(tt.obj)
			}
		})
	}
}
