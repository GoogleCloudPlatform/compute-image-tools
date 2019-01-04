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
	"cloud.google.com/go/compute/metadata"
	"context"
	"fmt"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// ParseWorkflow parses Daisy workflow file and returns Daisy workflow object or error in case of failure
func ParseWorkflow(ctx context.Context, path string, varMap map[string]string, project, zone, gcsPath, oauth, dTimeout, cEndpoint string, disableGCSLogs, diableCloudLogs, disableStdoutLogs bool) (*daisy.Workflow, error) {
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
		return nil, fmt.Errorf("unknown workflow Var %q passed to Workflow %q", k, w.Name)
	}

	if project != "" {
		w.Project = project
	} else if w.Project == "" && metadata.OnGCE() {
		w.Project, err = metadata.ProjectID()
		if err != nil {
			return nil, err
		}
	}
	if zone != "" {
		w.Zone = zone
	} else if w.Zone == "" && metadata.OnGCE() {
		w.Zone, err = metadata.Zone()
		if err != nil {
			return nil, err
		}
	}
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
	if diableCloudLogs {
		w.DisableCloudLogging()
	}
	if disableStdoutLogs {
		w.DisableStdoutLogging()
	}

	return w, nil
}
