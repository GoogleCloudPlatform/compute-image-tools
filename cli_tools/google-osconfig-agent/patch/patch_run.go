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

package patch

import (
	"os"
	"time"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/golang/protobuf/jsonpb"
)

// Init starts the patch system.
func Init() {
	// Load current patch state off disk.
	pr, err := loadState(state)
	if err != nil {
		logger.Errorf("loadState error: %v", err)
	} else if pr != nil && !pr.Complete {
		tasker.Enqueue("Gather instance inventory", func() { patchRunner(pr) })
	}

	// go runWatcher()
}

type patchRun struct {
	Job *patchJob

	StartedAt, EndedAt time.Time `json:",omitempty"`
	Complete           bool
	Errors             []string `json:",omitempty"`
}

type patchJob struct {
	*osconfigpb.ReportPatchJobInstanceDetailsResponse
}

// MarshalJSON marchals a patchConfig using jsonpb.
func (j *patchJob) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	s, err := m.MarshalToString(j)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalJSON unmarshals a patchConfig using jsonpb.
func (j *patchJob) UnmarshalJSON(b []byte) error {
	return jsonpb.UnmarshalString(string(b), j)
}

func (r *patchRun) in() bool {
	return true
}

func (r *patchRun) run() (reboot bool) {
	logger.Debugf("run %s", r.Job.PatchJobName)

	r.StartedAt = time.Now()

	if r.Complete {
		return false
	}

	defer func() {
		if err := saveState(state, r); err != nil {
			logger.Errorf("saveState error: %v", err)
		}
	}()

	// Make sure we are still in the patch window.
	if !r.in() {
		logger.Debugf("%s not in patch window", r.Job.PatchJobName)
		return false
	}

	if err := saveState(state, r); err != nil {
		logger.Errorf("saveState error: %v", err)
		r.Errors = append(r.Errors, err.Error())
	}

	reboot, err := runUpdates(r.Job.PatchConfig)
	if err != nil {
		// TODO: implement retries
		logger.Errorf("runUpdates error: %v", err)
		r.Errors = append(r.Errors, err.Error())
		return false
	}

	// Make sure we are still in the patch window
	if !r.in() {
		logger.Errorf("%s timedout", r.Job.PatchJobName)
		r.Errors = append(r.Errors, "Patch window timed out")
		return false
	}

	if !reboot {
		r.Complete = true
		r.EndedAt = time.Now()
	}
	return reboot
}

func patchRunner(pr *patchRun) {
	logger.Debugf("patchrunner running %s", pr.Job.PatchJobName)
	reboot := pr.run()
	if pr.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_NEVER {
		return
	}
	if (pr.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_ALWAYS) ||
		(((pr.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_DEFAULT) ||
			(pr.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_REBOOT_CONFIG_UNSPECIFIED)) &&
			reboot) {
		logger.Debugf("reboot requested %s", pr.Job.PatchJobName)
		if err := rebootSystem(); err != nil {
			logger.Errorf("error running reboot: %s", err)
		} else {
			// Reboot can take a bit, shutdown the agent so other activities don't start.
			os.Exit(0)
		}
	}
	logger.Debugf("finished patch window %s", pr.Job.PatchJobName)
}
