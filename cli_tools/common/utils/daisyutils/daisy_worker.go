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
	"context"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/assert"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
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

// NewDaisyWorker returns an implementation of DaisyWorker. The returned value is
// designed to be run once and discarded. In other words, don't run the same instance twice.
//
// If modifiers doesn't include a resource labeler, one will be created.
func NewDaisyWorker(wf *daisy.Workflow, env EnvironmentSettings,
	logger logging.Logger, modifiers ...WorkflowModifier) DaisyWorker {
	modifiers = append(modifiers, &ApplyEnvToWorkflow{env}, &ConfigureDaisyLogging{env})
	if env.NoExternalIP {
		modifiers = append(modifiers, &RemoveExternalIPModifier{})
	}
	return &defaultDaisyWorker{wf: wf, env: env, logger: logger, modifiers: createResourceLabelerIfMissing(env, modifiers)}
}

// createResourceLabelerIfMissing checks whether there is a resource labeler in modifiers.
// If not, then it creates a new one.
func createResourceLabelerIfMissing(env EnvironmentSettings, modifiers []WorkflowModifier) []WorkflowModifier {
	for _, modifier := range modifiers {
		switch modifier.(type) {
		case *ResourceLabeler:
			return modifiers
		}
	}
	assert.NotEmpty(env.Tool.ResourceLabelName)
	assert.NotEmpty(env.ExecutionID)
	return append(modifiers, NewResourceLabeler(
		env.Tool.ResourceLabelName, env.ExecutionID, env.Labels, env.StorageLocation))
}

type defaultDaisyWorker struct {
	wf        *daisy.Workflow
	logger    logging.Logger
	env       EnvironmentSettings
	modifiers []WorkflowModifier
}

// Run runs the daisy workflow with the supplied vars.
func (w *defaultDaisyWorker) Run(vars map[string]string) error {
	if err := (&ApplyAndValidateVars{w.env, vars}).Modify(w.wf); err != nil {
		return err
	}
	for _, t := range w.modifiers {
		if err := t.Modify(w.wf); err != nil {
			return err
		}
	}
	err := RunWorkflowWithCancelSignal(context.Background(), w.wf)
	if w.wf.Logger != nil {
		for _, trace := range w.wf.Logger.ReadSerialPortLogs() {
			w.logger.Trace(trace)
		}
	}
	if err != nil {
		PostProcessDErrorForNetworkFlag(w.env.Tool.HumanReadableName, err, w.env.Network, w.wf)
	}
	return err
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
	for _, key := range keys {
		m[key] = w.wf.GetSerialConsoleOutputValue(key)
	}
	return m, err
}

func (w *defaultDaisyWorker) Cancel(reason string) bool {
	if w.wf != nil {
		w.wf.CancelWithReason(reason)
		return true
	}

	//indicate cancel was not performed
	return false
}
