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

package daisycommon

import (
	"context"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// To rebuild the mock for DaisyWorker, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../mocks/mock_daisy_worker.go

// DaisyWorker is a facade over daisy.Workflow to facilitate mocking.
type DaisyWorker interface {
	RunAndReadSerialValue(key string, vars map[string]string) (string, error)
	Cancel(reason string) bool
}

// NewDaisyWorker returns an implementation of DaisyWorker. The returned value is
// designed to be run once and discarded. In other words, don't run RunAndReadSerialValue
// twice on the same instance.
func NewDaisyWorker(wf *daisy.Workflow, env EnvironmentSettings, logger logging.Logger) DaisyWorker {
	worker := &defaultDaisyWorker{wf, env, logger}
	worker.env.ApplyToWorkflow(wf)
	worker.env.ApplyWorkerCustomizations(wf)
	daisy_utils.UpdateAllInstanceNoExternalIP(wf, env.NoExternalIP)
	return worker
}

type defaultDaisyWorker struct {
	wf     *daisy.Workflow
	env    EnvironmentSettings
	logger logging.Logger
}

// RunAndReadSerialValue runs the daisy workflow with the supplied vars, and returns the serial
// output value associated with the supplied key.
func (w *defaultDaisyWorker) RunAndReadSerialValue(key string, vars map[string]string) (string, error) {
	for k, v := range vars {
		w.wf.AddVar(k, v)
	}
	err := w.wf.Run(context.Background())
	if w.wf.Logger != nil {
		for _, trace := range w.wf.Logger.ReadSerialPortLogs() {
			w.logger.Trace(trace)
		}
	}
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
