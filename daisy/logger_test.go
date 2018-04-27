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
	entries []*logEntry
	mx      sync.Mutex
}

// StepInfo logs information for the workflow step.
func (l *MockLogger) StepInfo(w *Workflow, stepName string, format string, a ...interface{}) {
	entry := &logEntry{
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
	entry := &logEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		Message:        fmt.Sprintf(format, a...),
	}
	l.mx.Lock()
	defer l.mx.Unlock()
	l.entries = append(l.entries, entry)
}

// f flushes all loggers.
func (l *MockLogger) FlushAll() {}

func (l *MockLogger) getEntries() []*logEntry {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.entries[:]
}

func TestCreateLogger(t *testing.T) {
	ctx := context.Background()
	prj := "test-project"

	c, err := logging.NewClient(ctx, prj)
	if err != nil {
		t.Fatalf("Unabled to create test logging client.")
	}

	workflowWithoutCloudLoggingClient := New()
	workflowWithoutCloudLoggingClient.DisableCloudLogging()
	workflowWithoutCloudLoggingClient.createLogger(ctx)

	workflowWithoutGCSLoggingClient := New()
	workflowWithoutGCSLoggingClient.DisableGCSLogging()
	workflowWithoutGCSLoggingClient.createLogger(ctx)
	workflowWithoutGCSLoggingClient.cloudLoggingClient = c

	workflowWithAllLoggingClient := New()
	workflowWithAllLoggingClient.createLogger(ctx)
	workflowWithAllLoggingClient.cloudLoggingClient = c

	tests := []struct {
		desc                             string
		w                                *Workflow
		cloudLogging, gcsLogging, stdout bool
	}{
		{
			"with all logging",
			workflowWithAllLoggingClient,
			true, true, true,
		},
		{
			"without cloud logging",
			workflowWithoutCloudLoggingClient,
			false, true, true,
		},
		{
			"without gcs logging",
			workflowWithoutGCSLoggingClient,
			true, false, true,
		},
	}

	for _, tt := range tests {
		tt.w.createLogger(ctx)
		if (tt.w.Logger.(*daisyLog).cloudLogger == nil) == tt.cloudLogging {
			t.Errorf("%q: wanted cloud logger (%t), got %t", tt.desc, tt.cloudLogging, tt.w.Logger.(*daisyLog).cloudLogger != nil)
		}
		if (tt.w.Logger.(*daisyLog).gcsLogWriter == nil) == tt.gcsLogging {
			t.Errorf("%q: wanted gcs logger (%t), got %t", tt.desc, tt.gcsLogging, tt.w.Logger.(*daisyLog).gcsLogWriter != nil)
		}
		if tt.w.Logger.(*daisyLog).stdoutLogging != tt.stdout {
			t.Errorf("%q: wanted serial logging (%t), got %t", tt.desc, tt.stdout, tt.w.Logger.(*daisyLog).stdoutLogging)
		}
	}
}

func TestWriteWorkflowInfo(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.createLogger(context.Background())

	var b bytes.Buffer
	w.Logger.(*daisyLog).gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	w.Logger.WorkflowInfo(w, "test %s", "a")
	w.Logger.(*daisyLog).gcsLogWriter.Flush()

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
	w.createLogger(context.Background())

	var b bytes.Buffer
	w.Logger.(*daisyLog).gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	w.Logger.StepInfo(w, "StepName", "test %s", "a")
	w.Logger.FlushAll()

	got := b.String()
	want := "\\[Test\\]: \\d{4}/\\d{2}/\\d{2} \\d{2}:\\d{2}:\\d{2} [A-Z]{1,3} StepName: test a"
	match, _ := regexp.MatchString(want, got)
	if !match {
		t.Errorf("Wanted to match %s, got %s", want, got)
	}
}
