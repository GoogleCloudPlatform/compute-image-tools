//  Copyright 2018 Google Inc. All Rights Reserved.
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

package daisycommon

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// ParseWorkflow parses Daisy workflow file and returns Daisy workflow object or error in case of failure
func ParseWorkflow(path string, varMap map[string]string, project, zone, gcsPath, oauth, dTimeout, cEndpoint string, disableGCSLogs, disableCloudLogs, disableStdoutLogs bool) (*daisy.Workflow, error) {
	w, err := daisy.NewFromFile(path)
	if err != nil {
		return nil, err
	}
Loop:
	for k, v := range varMap {
		for wv := range w.Vars {
			if k == wv {
				w.AddVar(k, v)
				continue Loop
			}
		}
		return nil, daisy.Errf("unknown workflow Var %q passed to Workflow %q", k, w.Name)
	}

	SetWorkflowAttributes(w, WorkflowAttributes{
		Project:           project,
		Zone:              zone,
		GCSPath:           gcsPath,
		OAuth:             oauth,
		Timeout:           dTimeout,
		ComputeEndpoint:   cEndpoint,
		DisableGCSLogs:    disableGCSLogs,
		DisableCloudLogs:  disableCloudLogs,
		DisableStdoutLogs: disableStdoutLogs,
	})

	return w, nil
}

// WorkflowAttributes holds common attributes that are required to instantiate a daisy workflow.
type WorkflowAttributes struct {
	Project, Zone, GCSPath, OAuth, Timeout, ComputeEndpoint, WorkflowDirectory string
	DisableGCSLogs, DisableCloudLogs, DisableStdoutLogs, NoExternalIP          bool
}

// SetWorkflowAttributes sets workflow running attributes.
func SetWorkflowAttributes(w *daisy.Workflow, attrs WorkflowAttributes) {
	w.Project = attrs.Project
	w.Zone = attrs.Zone
	if attrs.GCSPath != "" {
		w.GCSPath = attrs.GCSPath
	}
	if attrs.OAuth != "" {
		w.OAuthPath = attrs.OAuth
	}
	if attrs.Timeout != "" {
		w.DefaultTimeout = attrs.Timeout
	}

	if attrs.ComputeEndpoint != "" {
		w.ComputeEndpoint = attrs.ComputeEndpoint
	}

	if attrs.DisableGCSLogs {
		w.DisableGCSLogging()
	}
	if attrs.DisableCloudLogs {
		w.DisableCloudLogging()
	}
	if attrs.DisableStdoutLogs {
		w.DisableStdoutLogging()
	}
}
