//  Copyright 2019 Google Inc. All Rights Reserved.
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

package logging

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
	"github.com/golang/protobuf/proto"
)

// LogWriter is a logger for interactive tools. It supports
// string messages and structured metrics.
//
// Structured metrics are accumulated over the lifespan of the logger.
//
// To rebuild the mock, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../../../mocks/mock_logger.go
type LogWriter interface {
	// WriteUser messages appear in the following places:
	//  1. Web UI and gcloud.
	//  2. Standard output of the CLI tool.
	//  3. Backend trace logs (all debug and user logs are combined to a single trace log).
	WriteUser(message string)
	// WriteDebug messages appear in the following places:
	//  1. Standard output of the CLI tool.
	//  2. Backend trace logs (all debug and user logs are combined to a single trace log).
	WriteDebug(message string)
	// WriteTrace messages are saved to the logging backend (OutputInfo.serial_outputs).
	WriteTrace(message string)
	// WriteMetric merges all non-default fields into a single OutputInfo instance.
	WriteMetric(metric *pb.OutputInfo)
}

// RedirectGlobalLogsToUser redirects the standard library's static logger
// to logWriter.User.
func RedirectGlobalLogsToUser(logWriter LogWriter) {
	log.SetPrefix("")
	log.SetFlags(0)
	log.SetOutput(redirectShim{logWriter})
}

// redirectShim forwards Go's standard logger to LogWriter.User.
type redirectShim struct {
	writer LogWriter
}

func (l redirectShim) Write(p []byte) (n int, err error) {
	l.writer.WriteUser(string(p))
	return len(p), nil
}

// LogReader exposes pb.OutputInfo to a consumer.
type LogReader interface {
	ReadOutputInfo() *pb.OutputInfo
}

// ToolLogger is a logger for interactive tools. It supports
// string messages and structured metrics.
//
// Structured metrics are accumulated over the lifespan of the logger.
type ToolLogger interface {
	LogWriter
	LogReader
}

// defaultToolLogger is an implementation of ToolLogger that writes to an arbitrary writer.
// It has the following behavior for each level:
//
// User:
//  - Writes to the underlying log.Logger with an optional prefix. The prefix is used by
//    gcloud and the web console for filtering which logs are shown to the user.
//  - In addition to writing to the underlying log.Logger, the messages are buffered for
//    inclusion in OutputInfo.SerialOutputs.
// Debug:
//  - Writes to the underlying log.Logger with an optional prefix.
//  - In addition to writing to the underlying log.Logger, the messages are buffered for
//    inclusion in OutputInfo.SerialOutputs.
// Trace:
//  - Included in OutputInfo.SerialOutputs
type defaultToolLogger struct {
	// userPrefix and debugPrefix are strings that are prepended to user and debug messages.
	// The userPrefix string should be kept in sync with the matcher used by gcloud and the
	// web UI when determining which log messages to show to the user.
	userPrefix, debugPrefix string

	// output: Destination for user and debug messages.
	output *log.Logger

	// trace: Buffer of trace messages. Cleared when ReadOutputInfo is called.
	trace []string

	// stdoutLogger: Buffer of messages sent to the output logger. Cleared when
	// ReadOutputInfo is called.
	userAndDebugBuffer strings.Builder

	// outputInfo: View of OutputInfo that is updated when WriteMetric is called.
	// Reset when ReadOutputInfo info is called.
	outputInfo *pb.OutputInfo

	// timestampFormat is the format string used when writing the current time in user and debug messages.
	timestampFormat string

	// timeProvider is a function that returns the current time. Typically time.Now. Exposed for testing.
	timeProvider func() time.Time

	// mutationLock should be taken when reading or writing trace, userAndDebugBuffer, or outputInfo.
	mutationLock sync.Mutex
}

// WriteUser writes message to the underlying log.Logger, and then buffers the message
// for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) WriteUser(message string) {
	l.write(l.userPrefix, message)
}

// WriteDebug writes message to the underlying log.Logger, and then buffers the message
// for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) WriteDebug(message string) {
	l.write(l.debugPrefix, message)
}

// WriteTrace buffers the message for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) WriteTrace(message string) {
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	l.trace = append(l.trace, message)
}

// WriteMetric keeps non-nil fields from metric for inclusion in ReadOutputInfo().
// Elements of list fields are appended to the underlying view.
func (l *defaultToolLogger) WriteMetric(metric *pb.OutputInfo) {
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	proto.Merge(l.outputInfo, metric)
}

// Returns a view comprised of:
//   - Calls to WriteMetric
//   - All user, debug, and trace logs. User and debug logs are appended into a single
//     member of OutputInfo.SerialLogs; each trace log is a separate member.
// All buffers are cleared when this is called. In other words, a subsequent call to
// ReadOutputInfo will return an empty object.
func (l *defaultToolLogger) ReadOutputInfo() *pb.OutputInfo {
	// Locking since ReadOutputInfo has a side effect of clearing the internal state.
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	// Prepend stdout logs (user, debug) to the trace logs.
	var combinedTrace []string
	if l.userAndDebugBuffer.Len() > 0 {
		combinedTrace = []string{l.userAndDebugBuffer.String()}
		l.userAndDebugBuffer.Reset()
	}
	ret := &pb.OutputInfo{SerialOutputs: append(combinedTrace, l.trace...)}
	l.trace = []string{}
	proto.Merge(ret, l.outputInfo)
	l.outputInfo = &pb.OutputInfo{}
	return ret
}

// write a message to the underlying logger, and buffer it for inclusion in OutputInfo.SerialLogs.
func (l *defaultToolLogger) write(prefix, message string) {
	var logLineBuilder strings.Builder
	// If there's a prefix, ensure it ends with a colon.
	if prefix != "" {
		logLineBuilder.WriteString(prefix)
		if !strings.HasSuffix(prefix, ":") {
			logLineBuilder.WriteByte(':')
		}
		logLineBuilder.WriteByte(' ')
	}
	logLineBuilder.WriteString(l.timeProvider().Format(l.timestampFormat))
	logLineBuilder.WriteByte(' ')
	logLineBuilder.WriteString(message)
	// Ensure the log always ends with a newline.
	if !strings.HasSuffix(message, "\n") {
		logLineBuilder.WriteByte('\n')
	}
	logLine := logLineBuilder.String()

	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()
	l.output.Print(logLine)
	l.userAndDebugBuffer.WriteString(logLine)
}

// NewToolLogger returns a ToolLogger that writes user and debug messages to
// stdout. The userPrefix string is prepended to WriteUser messages. Specify
// the string that gcloud and the web console uses to find its matches.
func NewToolLogger(userPrefix string) ToolLogger {
	return &defaultToolLogger{
		userPrefix:      userPrefix,
		debugPrefix:     "[debug]",
		timestampFormat: time.RFC3339,
		output:          log.New(os.Stdout, "", 0),
		trace:           []string{},
		outputInfo:      &pb.OutputInfo{},
		timeProvider:    time.Now,
	}
}
