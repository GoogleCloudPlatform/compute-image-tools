//  Copyright 2021 Google Inc. All Rights Reserved.
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

import daisy "github.com/GoogleCloudPlatform/compute-daisy"

// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../../../mocks/mock_workflow_hook.go

// WorkflowPreHook executes before the workflow runs.
type WorkflowPreHook interface {
	// PreRunHook allows a WorkflowHook to modify a workflow prior to running.
	PreRunHook(wf *daisy.Workflow) error
}

// WorkflowPostHook executes after the workflow runs.
type WorkflowPostHook interface {
	// PostRunHook allows a WorkflowHook to inspect the workflow's run error, and optionally
	// decide whether to retry the workflow, or to wrap the error to expose a more useful
	// error message.
	PostRunHook(err error) (wantRetry bool, wrapped error)
}
