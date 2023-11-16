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

	"github.com/golang/protobuf/proto"

	"github.com/GoogleCloudPlatform/compute-image-tools/proto/go/pb"
)

// Logger is a logger for CLI tools. It supports
// string messages and structured metrics.
//
// Structured metrics are accumulated over the lifespan of the logger.
//
// To rebuild the mock, run `go generate ./...`
//
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -destination ../../../mocks/mock_logger.go
type Logger interface {
	// User messages appear in the following places:
	//  1. Web UI and gcloud.
	//  2. Standard output of the CLI tool.
	//  3. Backend trace logs (all debug and user logs are combined to a single trace log).
	User(message string)
	// Debug messages appear in the following places:
	//  1. Standard output of the CLI tool.
	//  2. Backend trace logs (all debug and user logs are combined to a single trace log).
	Debug(message string)
	// Trace messages are saved to the logging backend (OutputInfo.serial_outputs).
	Trace(message string)
	// Metric merges all non-default fields into a single OutputInfo instance.
	Metric(metric *pb.OutputInfo)
}

// RedirectGlobalLogsToUser redirects the standard library's static logger.
// All messages are written to logger's User level.
func RedirectGlobalLogsToUser(logger Logger) {
	log.SetPrefix("")
	log.SetFlags(0)
	log.SetOutput(redirectShim{logger})
}

// redirectShim forwards Go's standard logger to Logger.User.
type redirectShim struct {
	writer Logger
}

func (l redirectShim) Write(p []byte) (n int, err error) {
	l.writer.User(string(p))
	return len(p), nil
}

// OutputInfoReader exposes pb.OutputInfo to a consumer.
type OutputInfoReader interface {
	ReadOutputInfo() *pb.OutputInfo
}

// ToolLogger implements Logger and OutputInfoReader. Create an instance at the
// start of a CLI tool's invocation, and pass that instance to dependencies that
// require logging.
type ToolLogger interface {
	// NewLogger creates a new logger that writes to this ToolLogger, but with a
	// different User prefix.
	NewLogger(userPrefix string) Logger
	Logger
	OutputInfoReader
}

// defaultToolLogger is an implementation of ToolLogger that writes to an arbitrary writer.
// It has the following behavior for each level:
//
// User:
//   - Writes to the underlying log.Logger with an optional prefix. The prefix is used by
//     gcloud and the web console for filtering which logs are shown to the user.
//   - In addition to writing to the underlying log.Logger, the messages are buffered for
//     inclusion in OutputInfo.SerialOutputs.
//
// Debug:
//   - Writes to the underlying log.Logger with an optional prefix.
//   - In addition to writing to the underlying log.Logger, the messages are buffered for
//     inclusion in OutputInfo.SerialOutputs.
//
// Trace:
//   - Included in OutputInfo.SerialOutputs
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

	// outputInfo: View of OutputInfo that is updated when Metric is called.
	// Reset when ReadOutputInfo info is called.
	outputInfo *pb.OutputInfo

	// timestampFormat is the format string used when writing the current time in user and debug messages.
	timestampFormat string

	// timeProvider is a function that returns the current time. Typically time.Now. Exposed for testing.
	timeProvider func() time.Time

	// mutationLock should be taken when reading or writing trace, userAndDebugBuffer, or outputInfo.
	mutationLock sync.Mutex
}

func (l *defaultToolLogger) NewLogger(userPrefix string) Logger {
	return &customPrefixLogger{userPrefix, l}
}

// User writes message to the underlying log.Logger, and then buffers the message
// for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) User(message string) {
	l.writeLine(l.userPrefix, message)
}

// Debug writes message to the underlying log.Logger, and then buffers the message
// for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) Debug(message string) {
	l.writeLine(l.debugPrefix, message)
}

// Trace buffers the message for inclusion in ReadOutputInfo().
func (l *defaultToolLogger) Trace(message string) {
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	l.trace = append(l.trace, message)
}

// Metric keeps non-nil fields from metric for inclusion in ReadOutputInfo().
// Elements of list fields are appended to the underlying view.
func (l *defaultToolLogger) Metric(metric *pb.OutputInfo) {
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	proto.Merge(l.outputInfo, metric)
}

// Returns a view comprised of:
//   - Calls to Metric
//   - All user, debug, and trace logs. User and debug logs are appended into a single
//     member of OutputInfo.SerialLogs; each trace log is a separate member.
//
// All buffers are cleared when this is called. In other words, a subsequent call to
// ReadOutputInfo will return an empty object.
func (l *defaultToolLogger) ReadOutputInfo() *pb.OutputInfo {
	// Locking since ReadOutputInfo has a side effect of clearing the internal state.
	l.mutationLock.Lock()
	defer l.mutationLock.Unlock()

	ret := l.buildOutput()
	proto.Merge(ret, l.outputInfo)
	l.resetBuffers()
	return ret
}

func (l *defaultToolLogger) resetBuffers() {
	l.userAndDebugBuffer.Reset()
	l.trace = []string{}
	l.outputInfo = &pb.OutputInfo{}
}

func (l *defaultToolLogger) buildOutput() *pb.OutputInfo {
	var combinedTrace []string
	if l.userAndDebugBuffer.Len() > 0 {
		combinedTrace = []string{l.userAndDebugBuffer.String()}
	}
	return &pb.OutputInfo{SerialOutputs: append(combinedTrace, l.trace...)}
}

// writeLine writes a message to the underlying logger, and buffer it for inclusion in OutputInfo.SerialLogs.
// A newline is added to message if not already present.
func (l *defaultToolLogger) writeLine(prefix, message string) {
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
// stdout. The userPrefix string is prepended to User messages. Specify
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

// customPrefixLogger is a Logger that writes to a ToolLogger using a custom prefix for User messages.
type customPrefixLogger struct {
	userPrefix string
	parent     *defaultToolLogger
}

func (s *customPrefixLogger) User(message string) {
	s.parent.writeLine(s.userPrefix, message)
}

func (s *customPrefixLogger) Debug(message string) {
	s.parent.Debug(message)
}

func (s *customPrefixLogger) Trace(message string) {
	s.parent.Trace(message)
}

func (s *customPrefixLogger) Metric(metric *pb.OutputInfo) {
	s.parent.Metric(metric)
}
