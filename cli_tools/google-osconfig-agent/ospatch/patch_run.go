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

package ospatch

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	osconfig "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/cloud.google.com/go/osconfig/apiv1alpha1"
	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/logger"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/tasker"
	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/status"
)

type patchStep string

const (
	identityTokenPath = "instance/service-accounts/default/identity?audience=osconfig.googleapis.com&format=full"

	unknown              = ""
	notified             = "notified"
	started              = "started"
	prePatchReboot       = "prePatchReboot"
	applyPatch           = "applyPatch"
	postPatchReboot      = "postPatchReboot"
	postPatchRebooted    = "postPatchRebooted"
	completeSuccess      = "completeSuccess"
	completeFailed       = "completeFailed"
	completeJobCompleted = "completeJobCompleted"
)

var (
	// These patch steps are ordered to allow us to skip steps that have already been completed. Progress should
	// always move forward, even if we need to retry a step.
	//
	// NOTE: Changing the name/string of these steps can break the agent. The numbers themselves can change
	// since we store the name as a string.
	//
	// TODO: Consider refactoring into generic "steps" which can be tested and retried in isolation.
	patchStepIndex = map[patchStep]int{
		unknown:              0,
		notified:             1,
		started:              2,
		prePatchReboot:       3,
		applyPatch:           4,
		postPatchReboot:      5,
		postPatchRebooted:    6, // even if we fail, we don't want to force reboot more than once.
		completeSuccess:      7,
		completeFailed:       8,
		completeJobCompleted: 9,
	}

	cancelC chan struct{}
)

func initPatch(ctx context.Context) {
	cancelC = make(chan struct{})
	disableAutoUpdates()
	go Run(ctx, cancelC)
}

// Configure manages the background patch service.
func Configure(ctx context.Context) {
	select {
	case _, ok := <-cancelC:
		if !ok && config.OSPatchEnabled() {
			// Patch currently stopped, reenable.
			logger.Debugf("Enabling OSPatch")
			initPatch(ctx)
		} else if ok && !config.OSPatchEnabled() {
			// This should never happen as nothing should be sending on this
			// channel.
			logger.Errorf("Someone sent on the cancelC channel, this should not have happened")
			close(cancelC)
		}
	default:
		if cancelC == nil && config.OSPatchEnabled() {
			// initPatch has not run yet.
			logger.Debugf("Enabling OSPatch")
			initPatch(ctx)
		} else if cancelC != nil && !config.OSPatchEnabled() {
			// Patch currently running, we need to stop it.
			logger.Debugf("Disabling OSPatch")
			close(cancelC)
		}
	}
}

// Run runs patching as a single blocking agent, use cancel to cancel.
func Run(ctx context.Context, cancel <-chan struct{}) {
	logger.Debugf("Running OSPatch background task.")
	savedPatchJobName := checkSavedState(ctx)
	watcher(ctx, savedPatchJobName, cancel, ackPatch)
	logger.Debugf("OSPatch background task stopping.")
}

// Load current patch state off disk, schedule the patch if it isn't complete,
// and return its name.
func checkSavedState(ctx context.Context) string {
	savedPatchJobName := ""
	pr, err := loadState(state)
	if err != nil {
		logger.Errorf("loadState error: %v", err)
	} else if pr != nil && !pr.Complete {
		savedPatchJobName = pr.Job.PatchJob
		logger.Infof("Loaded State, running patch: '%s'...", savedPatchJobName)
		tasker.Enqueue("Run patch", func() { patchRunner(ctx, pr) })
	}
	return savedPatchJobName
}

type patchRun struct {
	Job *patchJob

	StartedAt, EndedAt time.Time `json:",omitempty"`
	Complete           bool
	Errors             []string `json:",omitempty"`
	PatchStep          patchStep
	// TODO add Attempts and track number of retries with backoff, jitter, etc.
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

func (r *patchRun) saveState() (shouldStop bool) {
	if err := saveState(state, r); err != nil {
		logger.Errorf("saveState error: %v", err)
		return true
	}
	return false
}

func (r *patchRun) finishAndReportError(ctx context.Context, msg string) {
	r.Complete = true
	r.EndedAt = time.Now()
	r.PatchStep = completeFailed
	logger.Errorf(msg)
	if _, err := reportPatchDetails(ctx, r.Job.PatchJob, osconfigpb.Instance_FAILED, 0, msg); err != nil {
		logger.Errorf("Failed to report patch failure. Error: %v", err)
		return
	}
	r.saveState()
}

func (r *patchRun) finishJobComplete() {
	r.Complete = true
	r.EndedAt = time.Now()
	r.PatchStep = completeJobCompleted
	logger.Infof("PatchJob %s is complete. Canceling patch execution.", r.Job.PatchJob)
	r.saveState()
}

func (r *patchRun) reportState(ctx context.Context, patchState osconfigpb.Instance_PatchState) (shouldStop bool) {
	patchJob, err := reportPatchDetails(ctx, r.Job.PatchJob, patchState, 0, "")
	if err != nil {
		// If we fail to report state, we can't report that we failed.
		logger.Errorf("Failed to report state %s. Error: %v", patchState, err)
		return true
	}
	r.Job.ReportPatchJobInstanceDetailsResponse = patchJob
	if patchJob.PatchJobState == osconfigpb.ReportPatchJobInstanceDetailsResponse_COMPLETED {
		r.finishJobComplete()
		return true
	}
	r.saveState()
	return false
}

func (r *patchRun) rebootIfNeeded(ctx context.Context, postUpdate bool) (shouldStop bool) {
	if r.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_NEVER {
		return true
	}

	var reboot bool
	var err error
	if r.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_ALWAYS && postUpdate {
		reboot = true
		logger.Infof("PatchConfig dictates a reboot.")
	} else {
		reboot, err = systemRebootRequired()
		if err != nil {
			r.finishAndReportError(ctx, fmt.Sprintf("Error checking if a system reboot is required: %v", err))
			return true
		}
		if reboot {
			logger.Infof("System indicates a reboot is required.")
		} else {
			logger.Infof("System indicates a reboot is not required.")
		}
	}

	if reboot {
		if r.reportState(ctx, osconfigpb.Instance_REBOOTING) {
			return true
		}

		r.PatchStep = postPatchRebooted
		r.saveState()

		if r.Job.DryRun {
			logger.Infof("Dry run - not rebooting for patch job '%s'", r.Job.PatchJob)
		} else {
			err := rebootSystem()
			if err != nil {
				r.finishAndReportError(ctx, fmt.Sprintf("Failed to reboot system: %v", err))
				return true
			}

			// Reboot can take a bit, shutdown the agent so other activities don't start.
			os.Exit(0)
			return true
		}
	}
	return false
}

func (r *patchRun) reportSucceeded(ctx context.Context) {
	isFinalRebootRequired, err := systemRebootRequired()
	if err != nil {
		r.finishAndReportError(ctx, fmt.Sprintf("Unable to check if reboot is required: %v", err))
		return
	}

	r.Complete = true
	r.EndedAt = time.Now()

	finalState := osconfigpb.Instance_SUCCEEDED
	if isFinalRebootRequired {
		finalState = osconfigpb.Instance_SUCCEEDED_REBOOT_REQUIRED
	}

	if r.reportState(ctx, finalState) {
		return
	}
}

/**
 * Runs a patch from start to finish. Sometimes this happens in a single invocation. Other times
 * we need to handle the following edge cases:
 * - The watcher has initiated this multiple times for the same patch job.
 * - We have a saved state and are continuing after a reboot.
 * - An error occurred and we do another attempt starting where we last failed.
 * - The process was unexpectedly restarted and we are continuing from where we left off.
 */
func (r *patchRun) runPatch(ctx context.Context) {
	savedPatchState, err := loadState(state)
	if err != nil {
		logger.Errorf("loadState error: %v", err)
		return
	}

	if savedPatchState != nil && savedPatchState.Job.PatchJob == r.Job.PatchJob {
		// continue from previous patch step
		r.PatchStep = savedPatchState.PatchStep
	} else {
		// We have no saved state for this patch.
		r.PatchStep = started
	}

	if patchStepIndex[r.PatchStep] <= patchStepIndex[started] {
		logger.Debugf("Starting patchJob %s", r.Job)
		r.StartedAt = time.Now()

		if r.reportState(ctx, osconfigpb.Instance_STARTED) {
			return
		}
	}

	if patchStepIndex[r.PatchStep] <= patchStepIndex[prePatchReboot] {
		// check if we need to reboot
		if r.rebootIfNeeded(ctx, false) {
			return
		}
	}

	if patchStepIndex[r.PatchStep] <= patchStepIndex[applyPatch] {
		if r.Job.DryRun {
			if r.reportState(ctx, osconfigpb.Instance_APPLYING_PATCHES) {
				return
			}
			logger.Infof("Dry run - No updates applied for patch job '%s'", r.Job.PatchJob)
		} else {
			err = runUpdates(ctx, r)
			if err != nil {
				r.finishAndReportError(ctx, fmt.Sprintf("Failed to apply patches: %v", err))
				return
			}
		}
	}

	if patchStepIndex[r.PatchStep] <= patchStepIndex[postPatchReboot] {
		// check if we need to reboot
		if r.rebootIfNeeded(ctx, true) {
			return
		}
	}

	if patchStepIndex[r.PatchStep] <= patchStepIndex[completeSuccess] {
		r.PatchStep = completeSuccess
		r.reportSucceeded(ctx)
	}
	logger.Debugf("Completed patchJob %s", r.Job)
}

func patchRunner(ctx context.Context, pr *patchRun) {
	logger.Debugf("Running patch job %s", pr.Job.PatchJob)
	pr.runPatch(ctx)
	logger.Debugf("Finished patch job %s", pr.Job.PatchJob)
}

func ackPatch(ctx context.Context, patchJobName string) {
	// TODO: Don't load the state off disk. This should be cached in memory with its access
	// blocked by a mutex.
	currentPatchJob, err := loadState(state)
	if err != nil {
		logger.Errorf("Unable to load state to ack notification. Error: %v", err)
		return
	}

	// Notify the server if we haven't yet. If we've already been notified about this Job,
	// the server may have inadvertantly notified us twice (at least once deliver) so we
	// can ignore it.
	if currentPatchJob == nil || currentPatchJob.Job.PatchJob != patchJobName {
		res, err := reportPatchDetails(ctx, patchJobName, osconfigpb.Instance_NOTIFIED, 0, "")
		if err != nil {
			logger.Errorf("reportPatchDetails Error: %v", err)
			return
		}
		tasker.Enqueue("Run patch", func() {
			patchRunner(ctx, &patchRun{Job: &patchJob{res}})
		})
	}
}

func reportPatchDetails(ctx context.Context, patchJobName string, patchState osconfigpb.Instance_PatchState, attemptCount int64, failureReason string) (*osconfigpb.ReportPatchJobInstanceDetailsResponse, error) {
	// TODO: add retries. Patching shouldn't continue if we can't talk to the server.
	logger.Debugf("Reporting patch details name:'%s', state:'%s', failReason:'%s'", patchJobName, patchState, failureReason)

	client, err := osconfig.NewClient(ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
	if err != nil {
		logger.Errorf("osconfig.NewClient Error: %v", err)
		return nil, err
	}
	defer client.Close()

	// This can't be cached.
	identityToken, err := metadata.Get(identityTokenPath)
	if err != nil {
		return nil, err
	}

	request := osconfigpb.ReportPatchJobInstanceDetailsRequest{
		Resource:         config.Instance(),
		InstanceSystemId: config.ID(),
		PatchJob:         patchJobName,
		InstanceIdToken:  identityToken,
		State:            patchState,
		AttemptCount:     attemptCount,
		FailureReason:    failureReason,
	}

	res, err := client.ReportPatchJobInstanceDetails(ctx, &request)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, fmt.Errorf("code: %q, message: %q, details: %q", s.Code(), s.Message(), s.Details())
		}
		return nil, err
	}
	return res, nil
}
