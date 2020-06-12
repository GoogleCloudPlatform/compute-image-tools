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
	"fmt"
	"log"
	"time"
)

// LogEntry encapsulates a single log entry.
type LogEntry struct {
	LocalTimestamp time.Time `json:"localTimestamp"`
	Message        string    `json:"message"`
}

func newLogEntry(message string) *LogEntry {
	return &LogEntry{LocalTimestamp: time.Now(), Message: message}
}

func (e *LogEntry) String() string {
	return fmt.Sprintf("%s %s", e.LocalTimestamp.Format("2006-01-02T15:04:05Z"), e.Message)
}

// LoggerInterface is logger abstraction
type LoggerInterface interface {
	Log(message string)
}

// NewStdoutLogger creates a new logger which uses prefix for all messages logged.
// All messages are sent to stdout.
func NewStdoutLogger(prefix string) LoggerInterface {
	return stdoutLogger{prefix: prefix}
}

type stdoutLogger struct{ prefix string }

// Log logs a message
func (l stdoutLogger) Log(message string) {
	fmt.Printf("%s %s\n", l.prefix, newLogEntry(message))
}

// NewDefaultLogger creates a new logger that sends all messages to the default logger.
func NewDefaultLogger() LoggerInterface {
	return defaultLogger{}
}

type defaultLogger struct{}

func (d defaultLogger) Log(message string) {
	log.Printf("%s\n", newLogEntry(message))
}
