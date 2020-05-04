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

package service

import (
	"bytes"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkflowLoggable_GetKeyValueAsInt64Slice(t *testing.T) {
	wf := daisy.Workflow{}
	wf.AddSerialConsoleOutputValue("gb", "1,2,3")
	loggable := WorkflowToLoggable(&wf)

	assert.Equal(t, []int64{1, 2, 3}, loggable.GetKeyValueAsInt64Slice("gb"))
	assert.Empty(t, loggable.GetKeyValueAsInt64Slice("not-there"))
}

func TestWorkflowLoggable_GetKeyValueAsKey(t *testing.T) {
	wf := daisy.Workflow{}
	wf.AddSerialConsoleOutputValue("hello", "world")
	loggable := WorkflowToLoggable(&wf)

	assert.Equal(t, "world", loggable.GetKeyValueAsString("hello"))
	assert.Empty(t, loggable.GetKeyValueAsString("not-there"))
}

func TestWorkflowLoggable_ReadSerialPortLogs(t *testing.T) {
	wf := daisy.Workflow{
		Logger: daisyLogger{serialLogs: []string{
			"log-a", "log-b",
		}},
	}
	loggable := WorkflowToLoggable(&wf)

	assert.Equal(t, []string{"log-a", "log-b"}, loggable.ReadSerialPortLogs())
}

func TestWorkflowLoggable_ReadSerialPortLogs_SupportsMissingLogger(t *testing.T) {
	wf := daisy.Workflow{}
	loggable := WorkflowToLoggable(&wf)

	assert.Empty(t, loggable.ReadSerialPortLogs())
}

func TestLiteralLoggable_GetKeyValueAsInt64Slice(t *testing.T) {
	loggable := literalLoggable{
		int64s: map[string][]int64{
			"gb": {1, 2, 3},
		},
	}

	assert.Equal(t, []int64{1, 2, 3}, loggable.GetKeyValueAsInt64Slice("gb"))
	assert.Empty(t, loggable.GetKeyValueAsInt64Slice("not-there"))
}

func TestLiteralLoggable_GetKeyValueAsKey(t *testing.T) {
	loggable := literalLoggable{
		strings: map[string]string{"hello": "world"},
	}

	assert.Equal(t, "world", loggable.GetKeyValueAsString("hello"))
	assert.Empty(t, loggable.GetKeyValueAsString("not-there"))
}

func TestLiteralLoggable_ReadSerialPortLogs(t *testing.T) {
	loggable := literalLoggable{
		serials: []string{"log-a", "log-b"},
	}

	assert.Equal(t, []string{"log-a", "log-b"}, loggable.ReadSerialPortLogs())
}

type daisyLogger struct {
	serialLogs []string
}

func (d daisyLogger) WriteLogEntry(e *daisy.LogEntry)                                          {}
func (d daisyLogger) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {}
func (d daisyLogger) Flush()                                                                   {}
func (d daisyLogger) ReadSerialPortLogs() []string {
	return d.serialLogs
}
