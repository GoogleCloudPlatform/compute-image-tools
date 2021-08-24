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

import (
	"bytes"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GoogleCloudPlatform/compute-image-tools/common/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func Test_ConfigureDaisyLogging_LeavesLoggingEnabled_ByDefault(t *testing.T) {
	wf := &daisy.Workflow{}
	assertLoggingEnabled(t, wf)
	assert.NoError(t, (&ConfigureDaisyLogging{EnvironmentSettings{}}).Modify(wf))
	assertLoggingEnabled(t, wf)
}

func Test_ConfigureDaisyLogging_DisablesLoggingOnWorkflow_IfSpecifiedInEnvironment(t *testing.T) {
	wf := &daisy.Workflow{}
	assertLoggingEnabled(t, wf)
	assert.NoError(t, (&ConfigureDaisyLogging{EnvironmentSettings{
		DisableGCSLogs:    true,
		DisableCloudLogs:  true,
		DisableStdoutLogs: true,
	}}).Modify(wf))
	assertLoggingDisabled(t, wf)
}

func Test_ConfigureDaisyLogging_AppliesPrivacyLogTag(t *testing.T) {
	var buffer bytes.Buffer
	wf := &daisy.Workflow{}
	wf.Logger = logging.AsDaisyLogger(log.New(&buffer, "", 0))
	assert.NoError(t, (&ConfigureDaisyLogging{EnvironmentSettings{}}).Modify(wf))
	wf.LogWorkflowInfo("message [Privacy->content<-Privacy] message")
	assert.Contains(t, buffer.String(), "message content message")
}

func assertLoggingDisabled(t *testing.T, wf *daisy.Workflow) {
	t.Helper()
	assertLoggingState(t, wf, false)
}

func assertLoggingEnabled(t *testing.T, wf *daisy.Workflow) {
	t.Helper()
	assertLoggingState(t, wf, true)
}

func assertLoggingState(t *testing.T, wf *daisy.Workflow, expectEnabled bool) {
	t.Helper()
	// ApplyToWorkflow calls methods to disable logging, which in turn updates private
	// fields on daisy.Workflow. This test inspects private fields directly
	// to validate that logging is disabled.
	privateLoggingFields := []string{"gcsLoggingDisabled", "stdoutLoggingDisabled", "cloudLoggingDisabled"}
	for _, fieldName := range privateLoggingFields {
		realValue := reflect.ValueOf(wf).Elem().FieldByName(fieldName)
		assert.Equal(t, !expectEnabled, realValue.Bool(), "field: %s", fieldName)
	}
}
