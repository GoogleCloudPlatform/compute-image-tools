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

//go:build !test
// +build !test

package compute

import (
	"cloud.google.com/go/compute/metadata"
	daisy "github.com/GoogleCloudPlatform/compute-daisy"
)

// MetadataGCE implements MetadataGCEInterface
type MetadataGCE struct{}

// OnGCE reports whether this process is running on Google Compute Engine.
func (m *MetadataGCE) OnGCE() bool {
	return metadata.OnGCE()
}

// Zone returns the current VM's zone, such as "us-central1-b".
func (m *MetadataGCE) Zone() (string, error) {
	zone, err := metadata.Zone()
	if err != nil {
		return zone, daisy.Errf("failed to get GCE zone from metadata: %v", err)
	}
	return zone, nil
}

// ProjectID returns the current instance's project ID string.
func (m *MetadataGCE) ProjectID() (string, error) {
	project, err := metadata.ProjectID()
	if err != nil {
		return project, daisy.Errf("failed to get GCE project id from metadata: %v", err)
	}
	return project, nil
}
