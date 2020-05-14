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

	SetWorkflowAttributes(w, project, zone, gcsPath, oauth, dTimeout, cEndpoint, disableGCSLogs, disableCloudLogs, disableStdoutLogs)

	return w, nil
}

// SetWorkflowAttributes sets workflow running attributes.
func SetWorkflowAttributes(w *daisy.Workflow, project, zone, gcsPath, oauth, dTimeout, cEndpoint string, disableGCSLogs, disableCloudLogs, disableStdoutLogs bool) {
	w.Project = project
	w.Zone = zone
	if gcsPath != "" {
		w.GCSPath = gcsPath
	}
	if oauth != "" {
		w.OAuthPath = oauth
	}
	if dTimeout != "" {
		w.DefaultTimeout = dTimeout
	}

	if cEndpoint != "" {
		w.ComputeEndpoint = cEndpoint
	}

	if disableGCSLogs {
		w.DisableGCSLogging()
	}
	if disableCloudLogs {
		w.DisableCloudLogging()
	}
	if disableStdoutLogs {
		w.DisableStdoutLogging()
	}
}
