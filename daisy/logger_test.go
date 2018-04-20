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

package daisy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/logging"
)

type MockLogger struct {
	entries []*LogEntry
	mx      sync.Mutex
}

// StepInfo logs information for the workflow step.
func (l *MockLogger) StepInfo(w *Workflow, stepName string, format string, a ...interface{}) {
	entry := &LogEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		StepName:       stepName,
		Message:        fmt.Sprintf(format, a...),
	}
	l.mx.Lock()
	defer l.mx.Unlock()
	l.entries = append(l.entries, entry)
}

// WorkflowInfo logs information for the workflow.
func (l *MockLogger) WorkflowInfo(w *Workflow, format string, a ...interface{}) {
	entry := &LogEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		Message:        fmt.Sprintf(format, a...),
	}
	l.mx.Lock()
	defer l.mx.Unlock()
	l.entries = append(l.entries, entry)
}

// FlushAll flushes all loggers.
func (l *MockLogger) FlushAll() {
	// nop
}

func (l *MockLogger) GetEntries() []*LogEntry {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.entries[:]
}

func TestCreateLogger(t *testing.T) {
	ctx := context.Background()
	prj := "test-project"

	workflowWithoutCloudLoggingClient := New()
	workflowWithCloudLoggingClient := New()
	c, err := logging.NewClient(ctx, prj)
	if err != nil {
		t.Errorf("Unabled to create test logging client.")
	}
	workflowWithCloudLoggingClient.cloudLoggingClient = c

	tests := []struct {
		w *Workflow
		cloudLogging, gcsLogging, stdout,
		wantCloudLogging, wantGcsLogging, wantStdout bool
	}{
		{
			workflowWithCloudLoggingClient,
			false, false, false,
			false, false, false,
		},
		{
			workflowWithCloudLoggingClient,
			true, true, true,
			true, true, true,
		},
		{
			workflowWithoutCloudLoggingClient,
			true, false, true,
			false, true, true,
		},
	}

	for _, tt := range tests {
		logger := CreateLogger(ctx, tt.w, tt.cloudLogging, tt.gcsLogging, tt.stdout)

		if (logger.cloudLogger == nil) == tt.wantCloudLogging {
			t.Errorf("Wanted cloud logger (%t), got %t", tt.wantCloudLogging, logger.cloudLogger == nil)
		}
		if (logger.gcsLogWriter == nil) == tt.wantGcsLogging {
			t.Errorf("Wanted gcs logger (%t), got %t", tt.wantGcsLogging, logger.gcsLogWriter == nil)
		}
		if logger.stdoutLogging != tt.wantStdout {
			t.Errorf("Wanted serial logging (%t), got %t", tt.wantStdout, logger.stdoutLogging)
		}
	}
}

func TestWriteWorkflowInfo(t *testing.T) {

	w := New()
	w.Name = "Test"
	logger := CreateLogger(context.Background(), w, false, false, false)

	var b bytes.Buffer
	logger.gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	logger.WorkflowInfo(w, "test %s", "a")
	logger.gcsLogWriter.Flush()

	got := b.String()
	want := "\\[Test\\]: \\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2} [A-Z]{1,3} test a"
	match, _ := regexp.MatchString(want, got)
	if !match {
		t.Errorf("Wanted to match %s, got %s", want, got)
	}
}

func TestWriteStepInfo(t *testing.T) {

	w := New()
	w.Name = "Test"
	logger := CreateLogger(context.Background(), w, false, false, false)

	var b bytes.Buffer
	logger.gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	logger.StepInfo(w, "StepName", "test %s", "a")
	logger.FlushAll()

	got := b.String()
	want := "\\[Test\\]: \\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2} [A-Z]{1,3} StepName: test a"
	match, _ := regexp.MatchString(want, got)
	if !match {
		t.Errorf("Wanted to match %s, got %s", want, got)
	}
}
