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
	"regexp"
	"strings"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

var (
	// Definition of resource path:
	//   https://cloud.google.com/compute/docs/reference/rest/v1/networks/get
	networkPattern = regexp.MustCompile(`^(?:projects/([^/]+)/)?(?:global/networks/)?([^/]+)$`)
)

// NetworkResource contains the name and project of a compute.Network resource.
type NetworkResource struct {
	Name, Project string
}

// String formats the NetworkResource as a GCP-style resource identifier.
func (r *NetworkResource) String() string {
	if r.Name == "" {
		return ""
	}
	var parts []string
	if r.Project != "" {
		parts = []string{"projects", r.Project}
	}
	return strings.Join(append(parts, "global", "networks", r.Name), "/")
}

// SplitNetworkResource creates a NetworkResource instance from a user-provided identifier for a network.
// It does not validate the individual fields.
func SplitNetworkResource(originalInput string) (*NetworkResource, error) {
	if originalInput == "" {
		return &NetworkResource{}, nil
	}
	parts := networkPattern.FindStringSubmatch(trimResourcePrefix(originalInput))
	if len(parts) == 0 {
		return nil, daisy.Errf("%q is not a valid network resource identifier. See %s for naming guidelines.",
			originalInput, "https://cloud.google.com/compute/docs/reference/rest/v1/networks/get")
	}
	return &NetworkResource{
		Name:    parts[2],
		Project: parts[1],
	}, nil
}
