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
	"errors"
	"fmt"
	"math"
	"math/rand"
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

	acked          = "acked"
	prePatchReboot = "prePatchReboot"
	patching       = "patching"
	complete       = "completed"
)

var (
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

	if err := loadState(config.PatchStateFile()); err != nil {
		logger.Errorf("loadState error: %v", err)
	}

	liveState.RLock()
	for _, pr := range liveState.PatchRuns {
		tasker.Enqueue("Run patch", pr.runPatch)
	}
	liveState.RUnlock()

	watcher(ctx, cancel, ackPatch)
	logger.Debugf("OSPatch background task stopping.")
}

type patchRun struct {
	ctx    context.Context
	client *osconfig.Client

	Job                *patchJob
	StartedAt, EndedAt time.Time `json:",omitempty"`
	Complete           bool
	Errors             []string `json:",omitempty"`
	PatchStep          patchStep
	RebootCount        int
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

func (r *patchRun) close() {
	if r.client != nil {
		r.client.Close()
	}
}

func (r *patchRun) setStep(step patchStep) error {
	r.PatchStep = step
	if err := saveState(); err != nil {
		return fmt.Errorf("error saving state: %v", err)
	}
	return nil
}

func (r *patchRun) handleErrorState(msg string, err error) {
	if err == errServerCancel {
		r.reportCanceledState()
	} else {
		r.reportFailedState(msg)
	}
}

func (r *patchRun) reportFailedState(msg string) {
	logger.Errorf(msg)
	if err := r.reportPatchDetails(osconfigpb.Instance_FAILED, 0, msg); err != nil {
		logger.Errorf("Failed to report patch failure: %v", err)
	}
}

func (r *patchRun) reportCanceledState() {
	logger.Infof("Canceling patch execution for PatchJob %q: %s", r.Job.PatchJob, errServerCancel)
	if err := r.reportPatchDetails(osconfigpb.Instance_FAILED, 0, errServerCancel.Error()); err != nil {
		logger.Errorf("Failed to report patch cancelation: %v", err)
	}
}

var errServerCancel = errors.New("service marked PatchJob as completed")

func (r *patchRun) reportContinuingState(patchState osconfigpb.Instance_PatchState) error {
	if err := r.reportPatchDetails(patchState, 0, ""); err != nil {
		return fmt.Errorf("error reporting state %s: %v", patchState, err)
	}
	if r.Job.PatchJobState == osconfigpb.ReportPatchJobInstanceDetailsResponse_COMPLETED {
		return errServerCancel
	}
	return saveState()
}

func (r *patchRun) complete() {
	liveState.removePatchRun(r)
	liveState.jobComplete(r.Job.PatchJob)
	if err := saveState(); err != nil {
		logger.Errorf("Error saving state: %v", err)
	}
	r.close()
}

// TODO: Add MaxRebootCount so we don't loop endlessly.

func (r *patchRun) prePatchReboot() error {
	return r.rebootIfNeeded(true)
}

func (r *patchRun) postPatchReboot() error {
	return r.rebootIfNeeded(false)
}

func (r *patchRun) rebootIfNeeded(prePatch bool) error {
	if r.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_NEVER {
		return nil
	}

	var reboot bool
	var err error
	if r.Job.PatchConfig.RebootConfig == osconfigpb.PatchConfig_ALWAYS && !prePatch && r.RebootCount == 0 {
		reboot = true
		logger.Infof("PatchConfig dictates a reboot.")
	} else {
		reboot, err = systemRebootRequired()
		if err != nil {
			return fmt.Errorf("error checking if a system reboot is required: %v", err)
		}
		if reboot {
			logger.Infof("System indicates a reboot is required.")
		} else {
			logger.Infof("System indicates a reboot is not required.")
		}
	}

	if !reboot {
		return nil
	}

	if err := r.reportContinuingState(osconfigpb.Instance_REBOOTING); err != nil {
		return err
	}

	if r.Job.DryRun {
		logger.Infof("Dry run - not rebooting for patch job '%s'", r.Job.PatchJob)
		return nil
	}

	r.RebootCount++
	if err := rebootSystem(); err != nil {
		return fmt.Errorf("failed to reboot system: %v", err)
	}

	// Reboot can take a bit, pause here so other activities don't start.
	for {
		logger.Debugf("Waiting for system reboot.")
		time.Sleep(1 * time.Minute)
	}
}

func (r *patchRun) createClient() error {
	if r.client == nil {
		var err error
		r.client, err = osconfig.NewClient(r.ctx, option.WithEndpoint(config.SvcEndpoint()), option.WithCredentialsFile(config.OAuthPath()))
		if err != nil {
			return fmt.Errorf("osconfig.NewClient Error: %v", err)
		}
	}
	return nil
}

/**
 * Runs a patch from start to finish. Sometimes this happens in a single invocation. Other times
 * we need to handle the following edge cases:
 * - The watcher has initiated this multiple times for the same patch job.
 * - We have a saved state and are continuing after a reboot.
 * - An error occurred and we do another attempt starting where we last failed.
 * - The process was unexpectedly restarted and we are continuing from where we left off.
 */
func (r *patchRun) runPatch() {
	logger.Debugf("Running patch job %s", r.Job.PatchJob)
	if err := r.createClient(); err != nil {
		logger.Errorf("Error creating osconfig client: %v", err)
	}
	defer r.complete()

	for {
		switch r.PatchStep {
		default:
			r.reportFailedState(fmt.Sprintf("unknown step: %q", r.PatchStep))
			return
		case acked:
			logger.Debugf("Starting patchJob %s", r.Job)
			r.StartedAt = time.Now()
			if err := r.setStep(prePatchReboot); err != nil {
				r.reportFailedState(fmt.Sprintf("Error saving agent step: %v", err))
			}

			if err := r.reportContinuingState(osconfigpb.Instance_STARTED); err != nil {
				r.handleErrorState(err.Error(), err)
				return
			}
		case prePatchReboot:
			// We do this now to avoid a reboot loop prior to patching.
			if err := r.setStep(patching); err != nil {
				r.reportFailedState(fmt.Sprintf("Error saving agent step: %v", err))
			}
			if err := r.prePatchReboot(); err != nil {
				r.handleErrorState(fmt.Sprintf("Error runnning prePatchReboot: %v", err), err)
				return
			}
		case patching:
			if r.Job.DryRun {
				if err := r.reportContinuingState(osconfigpb.Instance_APPLYING_PATCHES); err != nil {
					r.handleErrorState(err.Error(), err)
					return
				}
				logger.Infof("Dry run - No updates applied for patch job '%s'", r.Job.PatchJob)
			} else {
				if err := runUpdates(r); err != nil {
					r.handleErrorState(fmt.Sprintf("Failed to apply patches: %v", err), err)
					return
				}
			}
			if err := r.postPatchReboot(); err != nil {
				r.handleErrorState(fmt.Sprintf("Error runnning postPatchReboot: %v", err), err)
				return
			}
			// We have not rebooted so patching is complete.
			if err := r.setStep(complete); err != nil {
				r.reportFailedState(fmt.Sprintf("Error saving agent step: %v", err))
			}
		case complete:
			isFinalRebootRequired, err := systemRebootRequired()
			if err != nil {
				r.reportFailedState(fmt.Sprintf("Unable to check if reboot is required: %v", err))
				return
			}

			finalState := osconfigpb.Instance_SUCCEEDED
			if isFinalRebootRequired {
				finalState = osconfigpb.Instance_SUCCEEDED_REBOOT_REQUIRED
			}

			if err := r.reportPatchDetails(finalState, 0, ""); err != nil {
				logger.Errorf("Failed to report state %s. Error: %v", finalState, err)
			}
			logger.Debugf("Successfully completed patchJob %s", r.Job)
			return
		}
	}
}

func ackPatch(ctx context.Context, patchJobName string) {
	// Notify the server if we haven't yet. If we've already been notified about this Job,
	// the server may have inadvertantly notified us twice (at least once deliver) so we
	// can ignore it.
	if !liveState.alreadyAckedJob(patchJobName) {
		j := &patchJob{&osconfigpb.ReportPatchJobInstanceDetailsResponse{PatchJob: patchJobName}}
		pr := &patchRun{ctx: ctx, Job: j}
		if err := pr.createClient(); err != nil {
			logger.Errorf("Error creating osconfig client: %v", err)
		}
		if err := pr.reportPatchDetails(osconfigpb.Instance_ACKED, 0, ""); err != nil {
			logger.Errorf("reportPatchDetails Error: %v", err)
			pr.complete()
			return
		}
		pr.setStep(acked)
		tasker.Enqueue("Run patch", pr.runPatch)
	}
}

// retry tries to retry f for no more than maxRetryTime.
func retry(maxRetryTime time.Duration, desc string, f func() error) error {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	var tot time.Duration
	for i := 1; ; i++ {
		err := f()
		if err == nil {
			return nil
		}

		// Always increasing with some jitter, longest wait will be 5min.
		nf := math.Min(float64(i)*float64(i)+float64(rnd.Intn(i)), 300)
		ns := time.Duration(int(nf)) * time.Second
		tot += ns
		if tot < maxRetryTime {
			return err
		}

		logger.Debugf("Error %s, attempt %d, retrying in %s: %v", desc, i, ns, err)
		time.Sleep(ns)
	}
}

// reportPatchDetails tries to report patch details for 35m.
func (r *patchRun) reportPatchDetails(patchState osconfigpb.Instance_PatchState, attemptCount int64, failureReason string) error {
	err := retry(2100*time.Second, "reporting patch details", func() error {
		// This can't be cached.
		identityToken, err := metadata.Get(identityTokenPath)
		if err != nil {
			return err
		}

		request := osconfigpb.ReportPatchJobInstanceDetailsRequest{
			Resource:         config.Instance(),
			InstanceSystemId: config.ID(),
			PatchJob:         r.Job.PatchJob,
			InstanceIdToken:  identityToken,
			State:            patchState,
			AttemptCount:     attemptCount,
			FailureReason:    failureReason,
		}
		logger.Debugf("Reporting patch details request: %+v", request)

		res, err := r.client.ReportPatchJobInstanceDetails(r.ctx, &request)
		if err != nil {
			if s, ok := status.FromError(err); ok {
				return fmt.Errorf("code: %q, message: %q, details: %q", s.Code(), s.Message(), s.Details())
			}
			return err
		}
		r.Job.ReportPatchJobInstanceDetailsResponse = res
		return nil
	})
	if err != nil {
		return fmt.Errorf("error reporting patch details: %v", err)
	}
	return nil
}
