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

// Package logger logs messages as appropriate.
package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/config"
	"github.com/tarm/serial"

	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

var (
	port               = &serialPort{config.SerialLogPort()}
	cloudLoggingClient *logging.Client
	cloudLogger        *logging.Logger
)

// Init instantiates the logger.
func Init(ctx context.Context, project string) {
	msg := fmt.Sprintf("OSConfig Agent (version %s) Init", config.Version())
	Infof(msg)

	var err error
	cloudLoggingClient, err = logging.NewClient(ctx, project)
	if err != nil {
		Errorf(err.Error())
		return
	}

	// This automatically detects and associates with a GCE resource.
	cloudLogger = cloudLoggingClient.Logger("osconfig")
	if err := cloudLogger.LogSync(ctx, logging.Entry{Severity: logging.Info, Payload: map[string]string{"localTimestamp": now(), "message": msg}}); err != nil {
		// This means cloud logging is not working, so don't continue to try to log.
		cloudLogger = nil
		Errorf(err.Error())
		return
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			cloudLogger.Flush()
		}
	}()
}

// Close closes the logger.
func Close() {
	cloudLoggingClient.Close()
}

type serialPort struct {
	port string
}

func (s *serialPort) Write(b []byte) (int, error) {
	c := &serial.Config{Name: s.port, Baud: 115200}
	p, err := serial.OpenPort(c)
	if err != nil {
		return 0, err
	}
	defer p.Close()

	return p.Write(b)
}

// LogEntry encapsulates a single log entry.
type LogEntry struct {
	Message   string            `json:"message"`
	Labels    map[string]string `json:"-"`
	CallDepth int               `json:"-"`
}

type logEntry struct {
	LogEntry
	LocalTimestamp string `json:"localTimestamp"`

	severity logging.Severity
	source   *logpb.LogEntrySourceLocation
}

func (e logEntry) String() string {
	// INFO: 2006-01-02T15:04:05.999999Z07:00 file.go:82: This is a log message.
	return fmt.Sprintf("%s: %s %s:%d: %s", e.severity, e.LocalTimestamp, e.source.File, e.source.Line, e.Message)
}

func log(e *logEntry, out io.Writer) {
	if cloudLogger != nil {
		cloudLogger.Log(logging.Entry{Severity: e.severity, SourceLocation: e.source, Payload: e, Labels: e.Labels})
	}
	out.Write([]byte(strings.TrimSpace(e.String()) + "\n"))
}

func now() string {
	// RFC3339 with microseconds.
	return time.Now().Format("2006-01-02T15:04:05.999999Z07:00")
}

func caller(depth int) *logpb.LogEntrySourceLocation {
	// Add 2 to depth to account for this function and the imediate caller.
	depth = depth + 2
	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = "???"
		line = 0
	}

	return &logpb.LogEntrySourceLocation{File: filepath.Base(file), Line: int64(line), Function: runtime.FuncForPC(pc).Name()}
}

// Debug logs a debug entry.
func Debug(e LogEntry) {
	if !config.Debug() {
		return
	}

	le := &logEntry{LocalTimestamp: now(), LogEntry: e, severity: logging.Debug, source: caller(e.CallDepth)}
	log(le, io.MultiWriter(os.Stdout, port))
}

// Debugf logs debug information.
func Debugf(format string, v ...interface{}) {
	Debug(LogEntry{CallDepth: 2, Message: fmt.Sprintf(format, v...)})
}

// Info logs a general log entry.
func Info(e LogEntry) {
	le := &logEntry{LocalTimestamp: now(), LogEntry: e, severity: logging.Info, source: caller(e.CallDepth)}
	log(le, io.MultiWriter(os.Stdout, port))
}

// Infof logs general information.
func Infof(format string, v ...interface{}) {
	Info(LogEntry{CallDepth: 2, Message: fmt.Sprintf(format, v...)})
}

// Warning logs a warning entry.
func Warning(e LogEntry) {
	le := &logEntry{LocalTimestamp: now(), LogEntry: e, severity: logging.Warning, source: caller(e.CallDepth)}
	log(le, io.MultiWriter(os.Stderr, port))
}

// Warningf logs warning information.
func Warningf(format string, v ...interface{}) {
	Warning(LogEntry{CallDepth: 2, Message: fmt.Sprintf(format, v...)})
}

// Error logs an error entry.
func Error(e LogEntry) {
	le := &logEntry{LocalTimestamp: now(), LogEntry: e, severity: logging.Error, source: caller(e.CallDepth)}
	log(le, io.MultiWriter(os.Stderr, port))
}

// Errorf logs error information.
func Errorf(format string, v ...interface{}) {
	Error(LogEntry{CallDepth: 2, Message: fmt.Sprintf(format, v...)})
}

// Fatal logs an error entry and exits.
func Fatal(e LogEntry) {
	le := &logEntry{LocalTimestamp: now(), LogEntry: e, severity: logging.Critical, source: caller(e.CallDepth)}
	log(le, io.MultiWriter(os.Stderr, port))
	Close()
	os.Exit(1)
}

// Fatalf logs error information and exits.
func Fatalf(format string, v ...interface{}) {
	Fatal(LogEntry{CallDepth: 2, Message: fmt.Sprintf(format, v...)})
}
