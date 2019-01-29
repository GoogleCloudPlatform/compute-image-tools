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
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	api "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/api/option"
)

const (
	identityTokenPath = "instance/service-accounts/default/identity?audience=osconfig.googleapis.com&format=full"
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
	// TODO add Attempts and track number of retries with backoff, jitter, etc.
}

type patchJob struct {
	*api.ReportPatchJobInstanceDetailsResponse
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

func (r *patchRun) saveState() (shouldStop bool) {
	if err := saveState(state, r); err != nil {
		logger.Errorf("saveState error: %v", err)
		return true
	}
	return false
}

func (r *patchRun) finishAndReportError(ctx context.Context, msg string, err error) {
	r.Complete = true
	r.EndedAt = time.Now()
	errMsg := fmt.Sprintf(msg, err)
	logger.Errorf(errMsg)
	_, rptErr := reportPatchDetails(ctx, r.Job.PatchJobName, api.Instance_FAILED, 0, errMsg)
	if rptErr != nil {
		logger.Errorf("Failed to report patch failure. Error: %v", rptErr)
		return
	}
	r.saveState()
}

func (r *patchRun) finishJobComplete() {
	r.Complete = true
	r.EndedAt = time.Now()
	logger.Infof("PatchJob %s is complete. Canceling patch execution.", r.Job.PatchJobName)
	r.saveState()
}

func (r *patchRun) reportState(ctx context.Context, patchState api.Instance_PatchState) (shouldStop bool) {
	patchJob, err := reportPatchDetails(ctx, r.Job.PatchJobName, patchState, 0, "")
	if err != nil {
		// If we fail to report state, we can't report that we failed.
		logger.Errorf("Failed to report state %s. Error: %v", patchState, err)
		return true
	}
	r.Job.ReportPatchJobInstanceDetailsResponse = patchJob
	if patchJob.PatchJobState == api.ReportPatchJobInstanceDetailsResponse_COMPLETED {
		r.finishJobComplete()
		return true
	}
	r.saveState()
	return false
}

func (r *patchRun) rebootIfNeeded(ctx context.Context, postUpdate bool) (shouldStop bool) {
	if r.Job.PatchConfig.RebootConfig == api.PatchConfig_NEVER {
		return true
	}

	var reboot bool
	var err error
	if r.Job.PatchConfig.RebootConfig == api.PatchConfig_ALWAYS && postUpdate {
		reboot = true
	} else {
		reboot, err = systemRebootRequired()
		if err != nil {
			r.finishAndReportError(ctx, "Unable to check if reboot is required: %v", err)
			return true
		}
	}

	if reboot {
		if r.reportState(ctx, api.Instance_REBOOTING) {
			return true
		}

		err := rebootSystem()
		if err != nil {
			r.finishAndReportError(ctx, "Failed to reboot system: %v", err)
			return true
		}
		// Reboot can take a bit, shutdown the agent so other activities don't start.
		os.Exit(0)
		return true
	}
	return false
}

func (r *patchRun) reportSucceeded(ctx context.Context) {
	isFinalRebootRequired, err := systemRebootRequired()
	if err != nil {
		r.finishAndReportError(ctx, "Unable to check if reboot is required: %v", err)
		return
	}

	r.Complete = true
	r.EndedAt = time.Now()

	finalState := api.Instance_SUCCEEDED
	if isFinalRebootRequired {
		finalState = api.Instance_REBOOT_REQUIRED
	}

	if r.reportState(ctx, finalState) {
		return
	}
}

func (r *patchRun) runPatch(ctx context.Context) {

	savedPatchState, err := loadState(state)
	if err != nil {
		logger.Errorf("loadState error: %v", err)
		return
	} else if savedPatchState != nil && savedPatchState.Complete {
		logger.Infof("PatchJob already complete. %s", savedPatchState.Job.PatchJobName)
		return
	}

	logger.Debugf("Running patchJob %s", r.Job)
	r.StartedAt = time.Now()

	if r.reportState(ctx, api.Instance_STARTED) {
		return
	}

	// check if we need to reboot
	if r.rebootIfNeeded(ctx, false) {
		return
	}

	if r.reportState(ctx, api.Instance_APPLYING_PATCHES) {
		return
	}

	err = runUpdates(r.Job.PatchConfig)
	if err != nil {
		r.finishAndReportError(ctx, "Failed to apply patches: %v", err)
		return
	}

	// check if we need to reboot
	if r.rebootIfNeeded(ctx, true) {
		return
	}

	r.reportSucceeded(ctx)
}

func patchRunner(ctx context.Context, pr *patchRun) {
	logger.Debugf("Running patch job %s", pr.Job.PatchJobName)
	pr.runPatch(ctx)
	logger.Debugf("Finished patch job %s", pr.Job.PatchJobName)
}

func ackPatch(ctx context.Context, patchJobName string) {
	res, err := reportPatchDetails(ctx, patchJobName, api.Instance_STARTED, 0, "")
	if err != nil {
		logger.Errorf("reportPatchDetails Error: %v", err)
		return
	}

	tasker.Enqueue("Run patch", func() { patchRunner(ctx, &patchRun{Job: &patchJob{res}}) })
}

func reportPatchDetails(ctx context.Context, patchJobName string, patchState api.Instance_PatchState, attemptCount int64, failureReason string) (*api.ReportPatchJobInstanceDetailsResponse, error) {
	// TODO: add retries. Patching shouldn't continue if we can't talk to the server.

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		logger.Errorf("osconfig.NewClient Error: %v", err)
		return nil, err
	}

	identityToken, err := metadata.Get(identityTokenPath)
	if err != nil {
		return nil, err
	}

	fullInstanceName, err := config.Instance()
	if err != nil {
		return nil, err
	}

	instanceID, err := metadata.InstanceID()
	if err != nil {
		return nil, err
	}

	request := api.ReportPatchJobInstanceDetailsRequest{
		Resource:         fullInstanceName,
		InstanceSystemId: instanceID,
		PatchJobName:     patchJobName,
		InstanceIdToken:  identityToken,
		State:            patchState,
		AttemptCount:     attemptCount,
		FailureReason:    failureReason,
	}

	res, err := client.ReportPatchJobInstanceDetails(ctx, &request)
	return res, err
}
