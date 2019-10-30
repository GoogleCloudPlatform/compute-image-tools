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

package service

import (
	"flag"
	"os"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/param"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// CliToolWithLogging abstracts the interface of a cli tool with logging. A tool
// which implemented this interface adopts log service automatically.
type CliToolWithLogging interface {
	// ActionType indicates the action type of the tool. It represents which tool
	// function this log line is about. "ImageImport" is an example.
	ActionType() ActionType

	// InitParamLog initializes tool's input params in order to be logged
	InitParamLog() InputParams

	// Run is the entry function of the tool
	Run() (*daisy.Workflow, *param.UpdatedParams, error)
}

// RunCliToolWithLogging runs the cli tool with server logging
func RunCliToolWithLogging(t CliToolWithLogging) {
	flag.Parse()
	paramLog := t.InitParamLog()
	l := NewLoggingServiceLogger(t.ActionType(), paramLog)
	if _, err := l.runWithServerLogging(t.Run); err != nil {
		os.Exit(1)
	}
}
