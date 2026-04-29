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

func TestSplitNetworkResource_ValidCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *NetworkResource
	}{
		{
			name:     "empty",
			input:    "",
			expected: &NetworkResource{},
		}, {
			name:  "URL with version and https",
			input: "https://compute.googleapis.com/compute/v1/projects/project-id/global/networks/network-id",
			expected: &NetworkResource{
				Name:    "network-id",
				Project: "project-id",
			},
		}, {
			name:  "URL with no version without https",
			input: "//compute.googleapis.com/compute/projects/project-id/global/networks/network-id",
			expected: &NetworkResource{
				Name:    "network-id",
				Project: "project-id",
			},
		}, {
			name:  "path only",
			input: "projects/project-id/global/networks/network-id",
			expected: &NetworkResource{
				Name:    "network-id",
				Project: "project-id",
			},
		}, {
			name:  "no project portion",
			input: "global/networks/net-id",
			expected: &NetworkResource{
				Name: "net-id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := SplitNetworkResource(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSplitNetworkResource_InvalidCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "require global if missing project",
			input: "networks/net-id",
		}, {
			name:  "require projects prefix project portion",
			input: "project-id/global/networks/net-id",
		}, {
			name:  "URL with version without https",
			input: "//compute.googleapis.com/compute/v1/projects/project-id/global/networks/network-id",
		}, {
			name:  "URL with no version and https",
			input: "https://compute.googleapis.com/compute/projects/project-id/global/networks/network-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitNetworkResource(tt.input)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "is not a valid network resource identifier")
		})
	}
}

func TestNetworkResource_String(t *testing.T) {
	tests := []struct {
		input    *NetworkResource
		expected string
	}{
		{
			input:    &NetworkResource{},
			expected: "",
		}, {
			input:    &NetworkResource{Project: "project-id"},
			expected: "",
		}, {
			input:    &NetworkResource{"network-id", "project-id"},
			expected: "projects/project-id/global/networks/network-id",
		}, {
			input:    &NetworkResource{Name: "network-id"},
			expected: "global/networks/network-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}
