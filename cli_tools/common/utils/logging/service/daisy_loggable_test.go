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
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"github.com/stretchr/testify/assert"
)

func TestNewLoggableFromWorkflow_ReturnsNilWhenWorkflowNil(t *testing.T) {
	assert.Nil(t, NewLoggableFromWorkflow(nil))
}

func TestWorkflowToLoggable_GetValueAsInt64Slice(t *testing.T) {
	wf := daisy.Workflow{}
	wf.AddSerialConsoleOutputValue("gb", "1,2,3")
	loggable := NewLoggableFromWorkflow(&wf)

	assert.Equal(t, []int64{1, 2, 3}, loggable.GetValueAsInt64Slice("gb"))
	assert.Empty(t, loggable.GetValueAsInt64Slice("not-there"))
}

func TestWorkflowToLoggable_GetValue(t *testing.T) {
	wf := daisy.Workflow{}
	wf.AddSerialConsoleOutputValue("hello", "world")
	loggable := NewLoggableFromWorkflow(&wf)

	assert.Equal(t, "world", loggable.GetValue("hello"))
	assert.Empty(t, loggable.GetValue("not-there"))
}

func TestWorkflowToLoggable_ReadSerialPortLogs(t *testing.T) {
	wf := daisy.Workflow{
		Logger: daisyLogger{serialLogs: []string{
			"log-a", "log-b",
		}},
	}
	loggable := NewLoggableFromWorkflow(&wf)

	assert.Equal(t, []string{"log-a", "log-b"}, loggable.ReadSerialPortLogs())
}

func TestWorkflowToLoggable_ReadSerialPortLogs_SupportsMissingDaisyLogger(t *testing.T) {
	wf := daisy.Workflow{}
	loggable := NewLoggableFromWorkflow(&wf)

	assert.Empty(t, loggable.ReadSerialPortLogs())
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
