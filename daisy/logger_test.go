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
	"fmt"
	"regexp"
	"sync"
	"testing"

	"cloud.google.com/go/logging"
	"github.com/stretchr/testify/assert"
)

type MockLogger struct {
	entries        []*LogEntry
	mx             sync.Mutex
	serialPortLogs map[string]string
}

func (l *MockLogger) WriteSerialPortLogsToCloudLogging(w *Workflow, instance string) {
	// no-op
}

func (l *MockLogger) AppendSerialPortLogs(w *Workflow, instance string, logs string) {
	l.mx.Lock()
	defer l.mx.Unlock()
	if l.serialPortLogs == nil {
		l.serialPortLogs = map[string]string{}
	}
	l.serialPortLogs[instance] += logs
}

func (l *MockLogger) ReadSerialPortLogs() []string {
	var logs []string
	for _, log := range l.serialPortLogs {
		logs = append(logs, log)
	}
	return logs
}

func (l *MockLogger) WriteLogEntry(e *LogEntry) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.entries = append(l.entries, e)
}

// f flushes all loggers.
func (l *MockLogger) Flush() {}

func (l *MockLogger) getEntries() []*LogEntry {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.entries[:]
}

func TestWriteWorkflowInfo(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.Logger = newDaisyLogger(false)

	var b bytes.Buffer
	w.Logger.(*daisyLog).gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	w.LogWorkflowInfo("test %s", "a")
	w.Logger.(*daisyLog).gcsLogWriter.Flush()

	got := b.String()
	want := "\\[Test\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}([+-]\\d{2}:\\d{2})|Z test a"
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
	w.Logger = newDaisyLogger(false)

	var b bytes.Buffer
	w.Logger.(*daisyLog).gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(&b)}

	w.LogStepInfo("StepName", "StepType", "test %s", "a")
	w.Logger.Flush()

	got := b.String()
	want := "\\[Test.StepName\\]: \\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}([+-]\\d{2}:\\d{2})|Z StepType: test a"
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
	w.Logger = newDaisyLogger(false)
	cl := &MockCloudLogWriter{}
	w.Logger.(*daisyLog).cloudLogger = cl
	var buf bytes.Buffer
	for i := 0; i < 98*1024; i++ {
		w.Logger.AppendSerialPortLogs(w, "instance-name", "Serial output\n")
		buf.WriteString("Serial output\n")
	}

	w.Logger.WriteSerialPortLogsToCloudLogging(w, "instance-name")

	// We expect 14 entries
	if len(cl.entries) != 14 {
		t.Errorf("Wanted %d, got %d", 14, len(cl.entries))
	}

	assertLogOutput(t, w.Logger.ReadSerialPortLogs(),
		[]string{"Serial logs for instance: instance-name\n" + buf.String()})
}

func TestSendSerialPortLogsToCloudMultipleInstances(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.Logger = newDaisyLogger(false)
	cl := &MockCloudLogWriter{}
	w.Logger.(*daisyLog).cloudLogger = cl

	contentOfLogs := []string{
		"line1\nline2",
		"more log info\t",
	}

	instanceAnnotatedLogs := []string{
		"Serial logs for instance: instance-0\nline1\nline2",
		"Serial logs for instance: instance-1\nmore log info\t",
	}

	for i, log := range contentOfLogs {
		w.Logger.AppendSerialPortLogs(w, fmt.Sprintf("instance-%d", i), log)
	}

	assertLogOutput(t, w.Logger.ReadSerialPortLogs(), instanceAnnotatedLogs)
}

func TestSendSerialPortLogsToCloudDisabled(t *testing.T) {
	w := New()
	w.Name = "Test"
	w.Logger = newDaisyLogger(false)

	w.Logger.AppendSerialPortLogs(w, "instance-name", "Serial output\n")

	assert.Equal(t, len(w.Logger.ReadSerialPortLogs()), 0,
		"Don't retain logs if cloud logging disabled.")
}

func assertLogOutput(t *testing.T, actualLogs []string, expectedLogs []string) {
	if len(actualLogs) != len(expectedLogs) {
		t.Errorf("Expected %d serial logs. Found %d",
			len(expectedLogs), len(actualLogs))
	}

	for _, log := range expectedLogs {
		assert.Contains(t, actualLogs, log)
	}
}
