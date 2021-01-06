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

	EnvironmentSettings{
		Project:           project,
		Zone:              zone,
		GCSPath:           gcsPath,
		OAuth:             oauth,
		Timeout:           dTimeout,
		ComputeEndpoint:   cEndpoint,
		DisableGCSLogs:    disableGCSLogs,
		DisableCloudLogs:  disableCloudLogs,
		DisableStdoutLogs: disableStdoutLogs,
	}.ApplyToWorkflow(w)

	return w, nil
}

// EnvironmentSettings controls the resources that are used during tool execution.
type EnvironmentSettings struct {
	// Location of workflows
	WorkflowDirectory string

	// Fields from daisy.Workflow
	Project, Zone, GCSPath, OAuth, Timeout, ComputeEndpoint string
	DisableGCSLogs, DisableCloudLogs, DisableStdoutLogs     bool

	// Worker instance customizations
	Network, Subnet       string
	ComputeServiceAccount string
	NoExternalIP          bool
}

// ApplyWorkerCustomizations sets variables on daisy.Workflow that
// are used when creating worker instances.
func (env EnvironmentSettings) ApplyWorkerCustomizations(wf *daisy.Workflow) {
	wf.AddVar("network", env.Network)
	wf.AddVar("subnet", env.Subnet)
	if env.ComputeServiceAccount != "" {
		wf.AddVar("compute_service_account", env.ComputeServiceAccount)
	}
}

// ApplyToWorkflow sets fields on daisy.Workflow from the environment settings.
func (env EnvironmentSettings) ApplyToWorkflow(w *daisy.Workflow) {
	w.Project = env.Project
	w.Zone = env.Zone
	if env.GCSPath != "" {
		w.GCSPath = env.GCSPath
	}
	if env.OAuth != "" {
		w.OAuthPath = env.OAuth
	}
	if env.Timeout != "" {
		w.DefaultTimeout = env.Timeout
	}
	if env.ComputeEndpoint != "" {
		w.ComputeEndpoint = env.ComputeEndpoint
	}
	if env.DisableGCSLogs {
		w.DisableGCSLogging()
	}
	if env.DisableCloudLogs {
		w.DisableCloudLogging()
	}
	if env.DisableStdoutLogs {
		w.DisableStdoutLogging()
	}
}
