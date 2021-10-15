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

package daisyutils

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// ConfigureDaisyLogging is a WorkflowHook that configures Daisy's
// logging settings using the user's logging preferences, specified in EnvironmentSettings.
type ConfigureDaisyLogging struct {
	env EnvironmentSettings
}

// PreRunHook applies the user's logging preferences to a daisy workflow.
func (t *ConfigureDaisyLogging) PreRunHook(wf *daisy.Workflow) error {
	if t.env.DaisyLogLinePrefix != "" {
		wf.Name = t.env.DaisyLogLinePrefix
	}
	wf.SetLogProcessHook(RemovePrivacyLogTag)
	if t.env.DisableGCSLogs {
		wf.DisableGCSLogging()
	}
	if t.env.DisableCloudLogs {
		wf.DisableCloudLogging()
	}
	if t.env.DisableStdoutLogs {
		wf.DisableStdoutLogging()
	}
	return nil
}
