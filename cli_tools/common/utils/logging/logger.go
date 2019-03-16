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

// Logger is responsible for logging to stdout
type Logger struct {
	Prefix string
}

// LoggerInterface is logger abstraction
type LoggerInterface interface {
	Log(message string)
}

// NewLogger creates a new logger which uses prefix for all messages logged
func NewLogger(prefix string) *Logger {
	return &Logger{Prefix: prefix}
}

// Log logs a message
func (l *Logger) Log(message string) {
	fmt.Printf("%s %s\n", l.Prefix, newLogEntry(message))
}
