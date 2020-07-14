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

package logging

import (
	"bufio"
	"bytes"
	"io"
	"log"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// AsDaisyLogger returns a daisy.Logger that writes to a log.Logger.
func AsDaisyLogger(logger *log.Logger) daisy.Logger {
	return goLogToDaisyLog{logger: logger}
}

// AsWriter returns an io.Writer that writes to a log.Logger.
func AsWriter(logger *log.Logger) io.Writer {
	return goLogToIoWriter{logger: logger}
}

// goLogToDaisyLog is an implementation of daisy.Logger that writes to a log.Logger backend.
type goLogToDaisyLog struct {
	logger *log.Logger
}

func (l goLogToDaisyLog) WriteLogEntry(e *daisy.LogEntry) {
	l.logger.Printf(e.Message)
}

func (l goLogToDaisyLog) WriteSerialPortLogs(w *daisy.Workflow, instance string, buf bytes.Buffer) {
	// no-op
}

func (l goLogToDaisyLog) ReadSerialPortLogs() []string {
	// no-op
	return nil
}

func (l goLogToDaisyLog) Flush() {
	// no-op
}

// goLogToIoWriter is an implementation of io.Writer that writes to a log.Logger backend.
type goLogToIoWriter struct {
	logger *log.Logger
}

func (l goLogToIoWriter) Write(p []byte) (n int, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	for scanner.Scan() {
		l.logger.Println(scanner.Text())
	}
	return len(p), nil
}
