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
	"path"
	"regexp"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
)

// Logger is a helper that encapsulates the logging logic for Daisy.
type Logger interface {
	WriteLogEntry(e *LogEntry)
	// AppendSerialPortLogs appends a portion of serial port logs for a GCE instance.
	AppendSerialPortLogs(w *Workflow, instance string, logs string)
	// WriteSerialPortLogsToCloudLogging writes all of the collected logs for instance to cloud logging.
	WriteSerialPortLogsToCloudLogging(w *Workflow, instance string)
	// ReadSerialPortLogs returns all collected serial port logs, with one entry per instance.
	ReadSerialPortLogs() []string
	Flush()
}

type cloudLogWriter interface {
	Log(e logging.Entry)
	Flush() error
}

// daisyLog wraps the different logging mechanisms that can be used.
type daisyLog struct {
	gcsLogWriter    *syncedWriter
	cloudLogger     cloudLogWriter
	stdoutLogging   bool
	logCleanupRegex *regexp.Regexp
	// A map of instance name to its serial logs.
	serialLogs map[string]*bytes.Buffer
}

// createLogger builds a Logger.
func (w *Workflow) createLogger(ctx context.Context) {
	l := newDaisyLogger(!w.stdoutLoggingDisabled)

	if !w.gcsLoggingDisabled {
		gcsLogger := NewGCSLogger(ctx, w.StorageClient, w.bucket, path.Join(w.logsPath, "daisy.log"))
		l.gcsLogWriter = &syncedWriter{buf: bufio.NewWriter(gcsLogger)}
		periodicFlush(func() { l.gcsLogWriter.Flush() })
	}

	if !w.cloudLoggingDisabled && w.cloudLoggingClient != nil {
		// Verify we can communicate with the log service.
		if err := w.cloudLoggingClient.Ping(ctx); err != nil {
			l.WriteLogEntry(&LogEntry{
				LocalTimestamp: time.Now(),
				WorkflowName:   getAbsoluteName(w),
				Message:        fmt.Sprintf("Unable to send logs to the Cloud Logging service, not sending logs: %v", err),
			})
			w.cloudLoggingClient = nil
		} else {
			cloudLogName := fmt.Sprintf("daisy-%s-%s", w.Name, w.id)
			l.cloudLogger = w.cloudLoggingClient.Logger(cloudLogName)
			periodicFlush(func() { l.cloudLogger.Flush() })
		}
	}

	w.Logger = l

	w.addCleanupHook(func() DError {
		w.Logger.Flush()
		return nil
	})
}

func newDaisyLogger(stdOutLoggingEnabled bool) *daisyLog {
	return &daisyLog{
		stdoutLogging: stdOutLoggingEnabled,
		serialLogs:    map[string]*bytes.Buffer{},
	}
}

// LogStepInfo logs information for the workflow step.
func (w *Workflow) LogStepInfo(stepName, stepType, format string, a ...interface{}) {
	entry := &LogEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		StepName:       stepName,
		StepType:       stepType,
		Message:        fmt.Sprintf(format, a...),
		Type:           "Daisy",
	}
	w.logEntry(entry)
}

// LogWorkflowInfo logs information for the workflow.
func (w *Workflow) LogWorkflowInfo(format string, a ...interface{}) {
	entry := &LogEntry{
		LocalTimestamp: time.Now(),
		WorkflowName:   getAbsoluteName(w),
		Message:        fmt.Sprintf(format, a...),
	}
	w.logEntry(entry)
}

func (w *Workflow) logEntry(e *LogEntry) {
	//  Execute all log process hooks
	rw := w
	for rw != nil {
		if rw.logProcessHook != nil {
			e.Message = rw.logProcessHook(e.Message)
		}
		rw = rw.parent
	}

	w.Logger.WriteLogEntry(e)
}

// AppendSerialPortLogs collects a segment of serial port logs for an instance.
func (l *daisyLog) AppendSerialPortLogs(w *Workflow, instance string, logs string) {
	// Only collect serial port logs if the user has opted-in to cloud logging.
	if l.cloudLogger == nil {
		return
	}
	if _, hasBuffer := l.serialLogs[instance]; !hasBuffer {
		l.serialLogs[instance] = &bytes.Buffer{}
	}
	l.serialLogs[instance].WriteString(logs)
}

// WriteSerialPortLogsToCloudLogging writes the serial port logs for an instance to cloud logging.
func (l *daisyLog) WriteSerialPortLogsToCloudLogging(w *Workflow, instance string) {
	if l.cloudLogger == nil {
		return
	}

	if _, hasBuffer := l.serialLogs[instance]; !hasBuffer {
		return
	}
	logs := l.serialLogs[instance].Bytes()

	writeLog := func(data []byte) {
		entry := &LogEntry{
			LocalTimestamp: time.Now(),
			WorkflowName:   getAbsoluteName(w),
			Message:        fmt.Sprintf("Serial port output for instance %q", instance),
			SerialPort1:    string(data),
			Type:           "Daisy",
		}
		l.cloudLogger.Log(logging.Entry{Timestamp: entry.LocalTimestamp, Payload: entry})
	}

	// Write the output to cloud logging only after instance has stopped.
	// Type assertion check is needed for tests not to panic.
	// Split if output is too long for log entry (100K max, we leave a 2K buffer).
	if len(logs) <= 98*1024 {
		writeLog(logs)
		return
	}
	bs := bytes.SplitAfter(logs, []byte("\n"))
	var data []byte
	for _, b := range bs {
		if len(data)+len(b) > 98*1024 {
			writeLog(data)
			data = b
		} else {
			data = append(data, b...)
		}
	}
	writeLog(data)
}

func (l *daisyLog) ReadSerialPortLogs() []string {
	allLogs := make([]string, 0, len(l.serialLogs))
	for instance, log := range l.serialLogs {
		allLogs = append(allLogs, fmt.Sprintf("Serial logs for instance: %s\n%s", instance, log.Bytes()))
	}
	return allLogs
}

// Flush flushes all loggers.
func (l *daisyLog) Flush() {
	if l.gcsLogWriter != nil {
		l.gcsLogWriter.Flush()
	}

	if l.cloudLogger != nil {
		l.cloudLogger.Flush()
	}
}

// LogEntry encapsulates a single log entry.
type LogEntry struct {
	LocalTimestamp time.Time `json:"localTimestamp"`
	WorkflowName   string    `json:"workflow"`
	StepName       string    `json:"stepName,omitempty"`
	StepType       string    `json:"stepType,omitempty"`
	SerialPort1    string    `json:"serialPort1,omitempty"`
	Message        string    `json:"message"`
	Type           string    `json:"type"`
}

func (l *daisyLog) WriteLogEntry(e *LogEntry) {
	if l.cloudLogger != nil {
		l.cloudLogger.Log(logging.Entry{Timestamp: e.LocalTimestamp, Payload: e})
	}

	if l.gcsLogWriter != nil {
		l.gcsLogWriter.Write([]byte(e.String()))
	}

	if l.stdoutLogging {
		fmt.Print(e)
	}
}

type syncedWriter struct {
	buf *bufio.Writer
	mx  sync.Mutex
}

func (l *syncedWriter) Write(b []byte) (int, error) {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.buf.Write(b)
}

func (l *syncedWriter) Flush() error {
	l.mx.Lock()
	defer l.mx.Unlock()
	return l.buf.Flush()
}

// GCSLogger is a logger that writes to a GCS object.
type GCSLogger struct {
	client         *storage.Client
	bucket, object string
	buf            *bytes.Buffer
	ctx            context.Context
}

// NewGCSLogger creates a new GCSLogger.
func NewGCSLogger(ctx context.Context, client *storage.Client, bucket, object string) *GCSLogger {
	return &GCSLogger{client: client, bucket: bucket, object: object, ctx: ctx}
}

func (l *GCSLogger) Write(b []byte) (int, error) {
	if l.buf == nil {
		l.buf = new(bytes.Buffer)
	}
	l.buf.Write(b)
	wc := l.client.Bucket(l.bucket).Object(l.object).NewWriter(l.ctx)
	wc.ContentType = "text/plain"
	if _, err := wc.Write(l.buf.Bytes()); err != nil {
		return 0, err
	}
	if err := wc.Close(); err != nil {
		return 0, err
	}
	return len(b), nil
}

func periodicFlush(f func()) {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			f()
		}
	}()
}

func getAbsoluteName(w *Workflow) string {
	name := w.Name
	for parent := w.parent; parent != nil; parent = parent.parent {
		name = parent.Name + "." + name
	}
	return name
}

func (e *LogEntry) String() string {
	var prefix string
	if e.StepName != "" {
		prefix = fmt.Sprintf("%s.%s", e.WorkflowName, e.StepName)
	} else {
		prefix = e.WorkflowName
	}
	var msg string
	if e.StepType != "" {
		msg = fmt.Sprintf("%s: %s", e.StepType, e.Message)
	} else {
		msg = e.Message
	}

	timestamp := e.LocalTimestamp.Format(time.RFC3339)
	return fmt.Sprintf("[%s]: %s %s\n", prefix, timestamp, msg)
}
