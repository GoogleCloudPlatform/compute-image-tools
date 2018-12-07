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
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	logFlags = log.Ldate | log.Lmicroseconds | log.Lshortfile
)

var (
	debugLog   = log.New(os.Stdout, "DEBUG: ", logFlags)
	infoLog    = log.New(os.Stdout, "INFO: ", logFlags)
	warningLog = log.New(os.Stderr, "WARN: ", logFlags)
	errorLog   = log.New(os.Stderr, "ERROR: ", logFlags)
	debug      = flag.Bool("debug", false, "set debug log verbosity")
)

// Debugf logs debug information.
func Debugf(format string, v ...interface{}) {
	if *debug {
		debugLog.Output(2, fmt.Sprintf(format, v...))
	}
}

// Infof logs general information.
func Infof(format string, v ...interface{}) {
	infoLog.Output(2, fmt.Sprintf(format, v...))
}

// Warningf logs warning information.
func Warningf(format string, v ...interface{}) {
	warningLog.Output(2, fmt.Sprintf(format, v...))
}

// Errorf logs error information.
func Errorf(format string, v ...interface{}) {
	errorLog.Output(2, fmt.Sprintf(format, v...))
}

// Fatalf logs fatal error information and exits.
func Fatalf(format string, v ...interface{}) {
	errorLog.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}
