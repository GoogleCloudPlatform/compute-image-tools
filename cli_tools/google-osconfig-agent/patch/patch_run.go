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
	"context"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/api/option"
)

// Init starts the patch system.
func Init(ctx context.Context) {
	savedPatchName := ""
	// Load current patch state off disk.
	pr, err := loadState(state)
	if err != nil {
		logger.Errorf("loadState error: %v", err)
	} else if pr != nil {
		savedPatchName = pr.Job.PatchJobName
		if !pr.Complete {
			client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
			if err != nil {
				logger.Errorf("osconfig.NewClient Error: %v", err)
			} else {
				tasker.Enqueue("Run patch", func() { patchRunner(ctx, client, pr) })
			}
		}
	}

	go watcher(ctx, savedPatchName)
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

// MarshalJSON marshals a patchConfig using jsonpb.
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

	// TODO: Change this to be calls to ReportPatchJobInstanceDetails.
	if !r.in() {
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

	// TODO: Change this to be calls to ReportPatchJobInstanceDetails.
	if !r.in() {
		return false
	}

	if !reboot {
		r.Complete = true
		r.EndedAt = time.Now()
	}
	return reboot
}

func patchRunner(ctx context.Context, client *osconfig.Client, pr *patchRun) {
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

func ackPatch(ctx context.Context, id string) {
	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		logger.Errorf("osconfig.NewClient Error: %v", err)
		return
	}

	// TODO: Add all necessary bits into the API.
	res, err := client.ReportPatchJobInstanceDetails(ctx, &osconfigpb.ReportPatchJobInstanceDetailsRequest{PatchJobName: id, State: osconfigpb.Instance_NOTIFIED})
	if err != nil {
		logger.Errorf("osconfig.ReportPatchJobInstanceDetails Error: %v", err)
		return
	}

	tasker.Enqueue("Run patch", func() { patchRunner(ctx, client, &patchRun{Job: &patchJob{res}}) })
}
