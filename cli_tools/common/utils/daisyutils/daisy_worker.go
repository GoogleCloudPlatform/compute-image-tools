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

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// To rebuild the mock for DaisyWorker, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../../../mocks/mock_daisy_worker.go

// DaisyWorker is a facade over daisy.Workflow to facilitate mocking.
type DaisyWorker interface {
	Run(vars map[string]string) error
	RunAndReadSerialValue(key string, vars map[string]string) (string, error)
	Cancel(reason string) bool
}

// NewDaisyWorker returns an implementation of DaisyWorker. The returned value is
// designed to be run once and discarded. In other words, don't run RunAndReadSerialValue
// twice on the same instance.
func NewDaisyWorker(wf *daisy.Workflow, env EnvironmentSettings, logger logging.Logger, traversals ...WorkflowTraversal) DaisyWorker {
	traversals = append(traversals, &ApplyEnvToWorkflow{env}, &ConfigureDaisyLogging{env})
	if env.NoExternalIP {
		traversals = append(traversals, &RemoveExternalIPTraversal{})
	}
	return &defaultDaisyWorker{wf: wf, env: env, logger: logger, traversals: traversals}
}

type defaultDaisyWorker struct {
	wf         *daisy.Workflow
	logger     logging.Logger
	env        EnvironmentSettings
	traversals []WorkflowTraversal
}

// Run runs the daisy workflow with the supplied vars.
func (w *defaultDaisyWorker) Run(vars map[string]string) error {
	if err := (&ApplyAndValidateVars{w.env, vars}).Traverse(w.wf); err != nil {
		return err
	}
	for _, t := range w.traversals {
		if err := t.Traverse(w.wf); err != nil {
			return err
		}
	}
	err := w.wf.Run(context.Background())
	if w.wf.Logger != nil {
		for _, trace := range w.wf.Logger.ReadSerialPortLogs() {
			w.logger.Trace(trace)
		}
	}
	return err
}

// RunAndReadSerialValue runs the daisy workflow with the supplied vars, and returns the serial
// output value associated with the supplied key.
func (w *defaultDaisyWorker) RunAndReadSerialValue(key string, vars map[string]string) (string, error) {
	err := w.Run(vars)
	if err != nil {
		return "", err
	}
	return w.wf.GetSerialConsoleOutputValue(key), nil
}

func (w *defaultDaisyWorker) Cancel(reason string) bool {
	if w.wf != nil {
		w.wf.CancelWithReason(reason)
		return true
	}

	//indicate cancel was not performed
	return false
}
