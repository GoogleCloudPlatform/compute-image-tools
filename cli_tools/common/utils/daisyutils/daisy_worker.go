//  Copyright 2020 Google Inc. All Rights Reserved.
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
	"errors"
	"fmt"
	"sync"

	daisy "github.com/GoogleCloudPlatform/compute-daisy"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/assert"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
)

// To rebuild the mock for DaisyWorker, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../../../mocks/mock_daisy_worker.go

// DaisyWorker is a facade over daisy.Workflow to facilitate mocking.
type DaisyWorker interface {
	Run(vars map[string]string) error
	RunAndReadSerialValue(key string, vars map[string]string) (string, error)
	RunAndReadSerialValues(vars map[string]string, keys ...string) (map[string]string, error)
	Cancel(reason string) bool
}

// WorkflowProvider returns a new instance of a Daisy workflow.
type WorkflowProvider func() (*daisy.Workflow, error)

// NewDaisyWorker returns an implementation of DaisyWorker. The returned value is
// designed to be run once and discarded. In other words, don't run the same instance twice.
//
// hooks contains additional WorkflowPreHook or WorkflowPostHook instances. If hooks doesn't
// include a resource labeler, one will be created.
func NewDaisyWorker(wf WorkflowProvider, env EnvironmentSettings,
	logger logging.Logger, hooks ...interface{}) DaisyWorker {
	hooks = append(createResourceLabelerIfMissing(env, hooks),
		&ApplyEnvToWorkflow{env},
		&ConfigureDaisyLogging{env},
		&FallbackToPDStandard{logger: logger},
	)
	if env.NoExternalIP {
		hooks = append(hooks, &RemoveExternalIPHook{})
	}
	for _, hook := range hooks {
		switch hook.(type) {
		case WorkflowPreHook:
			continue
		case WorkflowPostHook:
			continue
		default:
			panic(fmt.Sprintf("%T must implement WorkflowPreHook and/or WorkflowPostHook", hook))
		}
	}
	return &defaultDaisyWorker{workflowProvider: wf, cancel: make(chan string, 1), env: env, logger: logger, hooks: hooks}
}

// createResourceLabelerIfMissing checks whether there is a resource labeler in hook.
// If not, then it creates a new one.
func createResourceLabelerIfMissing(env EnvironmentSettings, hooks []interface{}) []interface{} {
	for _, hook := range hooks {
		switch hook.(type) {
		case *ResourceLabeler:
			return hooks
		}
	}
	assert.NotEmpty(env.Tool.ResourceLabelName)
	assert.NotEmpty(env.ExecutionID)
	return append(hooks, NewResourceLabeler(
		env.Tool.ResourceLabelName, env.ExecutionID, env.Labels, env.StorageLocation))
}

type defaultDaisyWorker struct {
	workflowProvider WorkflowProvider
	finishedWf       *daisy.Workflow
	logger           logging.Logger
	env              EnvironmentSettings
	hooks            []interface{}

	cancel      chan string
	cancelGuard sync.Once
}

// Run runs the daisy workflow with the supplied vars.
func (w *defaultDaisyWorker) Run(vars map[string]string) (err error) {
	var wf *daisy.Workflow
	for attempts := 0; attempts <= 1; attempts++ {
		if wf, err = w.workflowProvider(); err != nil {
			break
		}
		if err = w.checkIfCancelled(wf); err != nil {
			break
		}
		var retryRequested bool
		retryRequested, err = w.runOnce(wf, vars)
		if err == nil || !retryRequested {
			break
		}
		w.logger.Debug(fmt.Sprintf("retryRequested=%v. err=%v", retryRequested, err))
	}
	w.finishedWf = wf
	return err
}

// checkIfCancelled determines whether the workflow has been cancelled internally,
// or whether a client of DaisyWorker has called cancel. If so, then a non-nil
// error is returned describing the cancellation.
func (w *defaultDaisyWorker) checkIfCancelled(wf *daisy.Workflow) (err error) {
	canceled := false
	reason := ""
	select {
	case reason = <-w.cancel:
		canceled = true
	case <-wf.Cancel:
		canceled = true
	default:
		break
	}
	if canceled {
		msg := "workflow canceled"
		if reason != "" {
			msg = fmt.Sprintf("%s: %s", msg, reason)
		}
		err = errors.New(msg)
	}
	return err
}

// runOnce applies vars to the workflow, runs hooks, and runs the workflow. The retry
// return value indicates whether a hook has requested a retry.
func (w *defaultDaisyWorker) runOnce(wf *daisy.Workflow, vars map[string]string) (retry bool, err error) {
	if err := (&ApplyAndValidateVars{w.env, vars}).PreRunHook(wf); err != nil {
		return false, err
	}
	for _, hook := range w.hooks {
		preHook, isPreHook := hook.(WorkflowPreHook)
		if isPreHook {
			if err := preHook.PreRunHook(wf); err != nil {
				return false, err
			}
		}
	}
	err = RunWorkflowWithCancelSignal(wf, w.cancel)
	if wf.Logger != nil {
		for _, trace := range wf.Logger.ReadSerialPortLogs() {
			w.logger.Trace(trace)
		}
	}
	if err != nil {
		PostProcessDErrorForNetworkFlag(w.env.Tool.HumanReadableName, err, w.env.Network, wf)
		for _, hook := range w.hooks {
			postHook, isPostHook := hook.(WorkflowPostHook)
			if isPostHook {
				wantRetry := false
				wantRetry, err = postHook.PostRunHook(err)
				retry = retry || wantRetry
			}
		}
	}
	return retry, err
}

// RunAndReadSerialValue runs the daisy workflow with the supplied vars, and returns the serial
// output value associated with the supplied key.
func (w *defaultDaisyWorker) RunAndReadSerialValue(key string, vars map[string]string) (string, error) {
	m, err := w.RunAndReadSerialValues(vars, key)
	return m[key], err
}

// RunAndReadSerialValues runs the daisy workflow with the supplied vars, and returns the serial
// output values associated with the supplied keys.
func (w *defaultDaisyWorker) RunAndReadSerialValues(vars map[string]string, keys ...string) (map[string]string, error) {
	err := w.Run(vars)
	m := map[string]string{}
	if w.finishedWf != nil {
		for _, key := range keys {
			m[key] = w.finishedWf.GetSerialConsoleOutputValue(key)
		}
	}
	return m, err
}

func (w *defaultDaisyWorker) Cancel(reason string) bool {
	// once.Do is required to ensure that additional calls
	// to Cancel won't write to a closed channel.
	w.cancelGuard.Do(
		func() {
			w.cancel <- reason
			close(w.cancel)
		},
	)
	return true
}
