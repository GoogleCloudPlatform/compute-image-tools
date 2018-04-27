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
)

type MockLogger struct {
	entries []*logEntry
	mx      sync.Mutex
}

// StepInfo logs information for the workflow step.
func (l *MockLogger) StepInfo(w *Workflow, stepName, stepType string, format string, a ...interface{}) {
	entry := &logEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		StepName:       stepName,
		StepType:       stepType,
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

func TestWriteWorkflowInfo(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.createLogger(context.Background())

	var b bytes.Buffer
	w.Logger.(*daisyLog).gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	w.Logger.WorkflowInfo(w, "test %s", "a")
	w.Logger.(*daisyLog).gcsLogWriter.Flush()

	got := b.String()
	want := "\\[Test\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}-\\d{2}:\\d{2} test a"
	match, err := regexp.MatchString(want, got)
	if err != nil {
		t.Fatal(err)
	}
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

	w.Logger.StepInfo(w, "StepName", "StepType", "test %s", "a")
	w.Logger.FlushAll()

	got := b.String()
	want := "\\[Test.StepName\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}-\\d{2}:\\d{2} StepType: test a"
	match, _ := regexp.MatchString(want, got)
	if !match {
		t.Errorf("Wanted to match %s, got %s", want, got)
	}
}
