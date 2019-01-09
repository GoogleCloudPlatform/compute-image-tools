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
	"github.com/golang/protobuf/jsonpb"
)

type patchRun struct {
	Policy *patchPolicy

	StartedAt, EndedAt time.Time `json:",omitempty"`
	Complete           bool
	Errors             []string `json:",omitempty"`
}

type patchPolicy struct {
	*osconfigpb.PatchPolicy
}

// MarshalJSON marchals a patchPolicy using jsonpb.
func (p *patchPolicy) MarshalJSON() ([]byte, error) {
	m := jsonpb.Marshaler{}
	s, err := m.MarshalToString(p)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// UnmarshalJSON unmarshals apatchPolicy using jsonpb.
func (p *patchPolicy) UnmarshalJSON(b []byte) error {
	return jsonpb.UnmarshalString(string(b), p)
}

func (r *patchRun) in() bool {
	return true
}

func (r *patchRun) run() (reboot bool) {
	logger.Debugf("run %s", r.Policy.Name)

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
		logger.Debugf("%s not in patch window", r.Policy.Name)
		return false
	}

	if err := saveState(state, r); err != nil {
		logger.Errorf("saveState error: %v", err)
		r.Errors = append(r.Errors, err.Error())
	}

	reboot, err := runUpdates(r.Policy)
	if err != nil {
		// TODO: implement retries
		logger.Errorf("runUpdates error: %v", err)
		r.Errors = append(r.Errors, err.Error())
		return false
	}

	// Make sure we are still in the patch window
	if !r.in() {
		logger.Errorf("%s timedout", r.Policy.Name)
		r.Errors = append(r.Errors, "Patch window timed out")
		return false
	}

	if !reboot {
		r.Complete = true
		r.EndedAt = time.Now()
	}
	return reboot
}

func patchRunner(pp *osconfigpb.PatchPolicy) {
	logger.Debugf("patchrunner running %s", pp.Name)
	pr := &patchRun{Policy: &patchPolicy{pp}}
	reboot := pr.run()
	if pp.RebootConfig == osconfigpb.PatchPolicy_NEVER {
		return
	}
	if (pp.RebootConfig == osconfigpb.PatchPolicy_ALWAYS) ||
		(((pp.RebootConfig == osconfigpb.PatchPolicy_DEFAULT) ||
			(pp.RebootConfig == osconfigpb.PatchPolicy_REBOOT_CONFIG_UNSPECIFIED)) &&
			reboot) {
		logger.Debugf("reboot requested %s", pp.Name)
		if err := rebootSystem(); err != nil {
			logger.Errorf("error running reboot: %s", err)
		} else {
			// Reboot can take a bit, shutdown the agent so other activities don't start.
			os.Exit(0)
		}
	}
	logger.Debugf("finished patch window %s", pp.Name)
}
