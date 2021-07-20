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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// SubnetResource contains the name, project, and region of a compute.Subnetwork resource.
type SubnetResource struct {
	Name, Project, Region string
}

// String formats the SubnetResource as a GCP-style resource identifier.
func (r *SubnetResource) String() string {
	if r.Name == "" {
		return ""
	}
	if r.Region == "" {
		return r.Name
	}
	if r.Project == "" {
		return fmt.Sprintf("regions/%s/subnetworks/%s", r.Region, r.Name)
	}
	return fmt.Sprintf("projects/%s/regions/%s/subnetworks/%s", r.Project, r.Region, r.Name)
}

// SplitSubnetResource creates a SubnetResource instance from a user-provided identifier for a subnet.
// It does not validate the individual fields.
func SplitSubnetResource(originalInput string) (*SubnetResource, error) {
	if originalInput == "" {
		return &SubnetResource{}, nil
	}
	parts := strings.Split(trimResourcePrefix(originalInput), "/")
	resource := &SubnetResource{}
	if len(parts) == 6 && parts[0] == "projects" {
		resource.Project = parts[1]
		parts = parts[2:]
	}
	if len(parts) == 4 && parts[0] == "regions" {
		resource.Region = parts[1]
		parts = parts[2:]
	}
	if len(parts) == 2 && parts[0] == "subnetworks" {
		parts = parts[1:]
	}
	if len(parts) != 1 {
		return nil, daisy.Errf("%q is not a valid subnet resource identifier. See %s for naming guidelines.",
			originalInput, "https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get")
	}
	resource.Name = parts[0]
	return resource, nil
}
