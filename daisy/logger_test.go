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

func (l *MockLogger) SendSerialPortLogsToCloud(w *Workflow, instance string, buf bytes.Buffer) {
	// nop
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
	want := "\\[Test\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(-\\d{2}:\\d{2})|Z test a"
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
	want := "\\[Test.StepName\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(-\\d{2}:\\d{2})|Z StepType: test a"
	match, _ := regexp.MatchString(want, got)
	if !match {
		t.Errorf("Wanted to match %s, got %s", want, got)
	}
}

type MockCloudLogWriter struct {
	entries []*logging.Entry
	mx      sync.Mutex
}

func (cl *MockCloudLogWriter) Log(e logging.Entry) {
	cl.mx.Lock()
	defer cl.mx.Unlock()
	cl.entries = append(cl.entries, &e)
}

func (cl *MockCloudLogWriter) Flush() error {
	return nil
}

func TestSendSerialPortLogsToCloud(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.Logger = &daisyLog{}
	cl := &MockCloudLogWriter{}
	w.Logger.(*daisyLog).cloudLogger = cl
	var buf bytes.Buffer
	for i := 0; i < 98*1024; i++ {
		buf.WriteString("Serial output\n")
	}

	w.Logger.SendSerialPortLogsToCloud(w, "instance-name", buf)

	if len(cl.entries) != 14 {
		t.Errorf("Wanted %d", len(cl.entries))
	}
}

func TestSendSerialPortLogsToCloudDisabled(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.Logger = &daisyLog{}
	var buf bytes.Buffer
	buf.WriteString("Serial output\n")

	w.Logger.SendSerialPortLogsToCloud(w, "instance-name", buf)

	// Nothing to verify. Nothing happened.
}
