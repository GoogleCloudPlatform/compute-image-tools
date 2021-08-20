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

func TestSplitSubnetResource_ValidCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *SubnetResource
	}{
		{
			name:     "empty",
			input:    "",
			expected: &SubnetResource{},
		}, {
			name:  "URL with version",
			input: "https://compute.googleapis.com/compute/v1/projects/project-id/regions/region-id/subnetworks/subnet-id",
			expected: &SubnetResource{
				Name:    "subnet-id",
				Project: "project-id",
				Region:  "region-id",
			},
		}, {
			name:  "URL without version",
			input: "//compute.googleapis.com/compute/projects/project-id/regions/region-id/subnetworks/subnet-id",
			expected: &SubnetResource{
				Name:    "subnet-id",
				Project: "project-id",
				Region:  "region-id",
			},
		}, {
			name:  "path only",
			input: "projects/project-id/regions/region-id/subnetworks/subnet-id",
			expected: &SubnetResource{
				Name:    "subnet-id",
				Project: "project-id",
				Region:  "region-id",
			},
		}, {
			name:  "name only with resource prefix",
			input: "subnetworks/subnet-id",
			expected: &SubnetResource{
				Name: "subnet-id",
			},
		}, {
			name:  "name only",
			input: "subnet-id",
			expected: &SubnetResource{
				Name: "subnet-id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := SplitSubnetResource(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestSplitSubnetResource_InvalidCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "extra unrecognized prefix",
			input: "extra/garbage/projects/project-id/regions/region-id/subnetworks/subnet-id",
		}, {
			name:  "missing project prefix",
			input: "project-id/regions/region-id/subnetworks/subnet-id",
		}, {
			name:  "missing region prefix",
			input: "region-id/subnetworks/subnet-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitSubnetResource(tt.input)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "is not a valid subnet resource identifier")
		})
	}
}

func TestSubnetResource_String(t *testing.T) {
	tests := []struct {
		input    *SubnetResource
		expected string
	}{
		{
			input:    &SubnetResource{},
			expected: "",
		}, {
			input:    &SubnetResource{Project: "project-id"},
			expected: "",
		}, {
			input:    &SubnetResource{Project: "project-id", Region: "region-id"},
			expected: "",
		}, {
			input:    &SubnetResource{Name: "subnet-id"},
			expected: "subnet-id",
		}, {
			input:    &SubnetResource{Name: "subnet-id", Project: "project-id"},
			expected: "subnet-id",
		}, {
			input:    &SubnetResource{Name: "subnet-id", Region: "region-id"},
			expected: "regions/region-id/subnetworks/subnet-id",
		}, {
			input:    &SubnetResource{Name: "subnet-id", Region: "region-id", Project: "project-id"},
			expected: "projects/project-id/regions/region-id/subnetworks/subnet-id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input.String())
		})
	}
}
