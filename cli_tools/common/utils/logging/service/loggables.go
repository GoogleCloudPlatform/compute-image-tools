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

package service

import (
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"strconv"
	"strings"
)

// WorkflowToLoggable is a shim for a daisy workflow, exposing only those
// fields that are pertinent to logging.
func WorkflowToLoggable(wf *daisy.Workflow) Loggable {
	return workflowLoggable{wf: wf}
}

type workflowLoggable struct {
	wf *daisy.Workflow
}

func (w workflowLoggable) GetKeyValueAsKey(key string) string {
	return w.wf.GetSerialConsoleOutputValue(key)
}

func (w workflowLoggable) GetKeyValueAsInt64Slice(key string) []int64 {
	return getInt64Values(w.wf.GetSerialConsoleOutputValue(key))
}

func (w workflowLoggable) ReadSerialPortLogs() []string {
	if w.wf.Logger != nil {
		logs := w.wf.Logger.ReadSerialPortLogs()
		view := make([]string, len(logs))
		copy(view, logs)
		return view
	}
	return nil
}

type literalLoggable struct {
	strings map[string]string
	int64s  map[string][]int64
	serials []string
}

func (w literalLoggable) GetKeyValueAsKey(key string) string { return w.strings[key] }

func (w literalLoggable) GetKeyValueAsInt64Slice(key string) []int64 { return w.int64s[key] }

func (w literalLoggable) ReadSerialPortLogs() []string { return w.serials }

func getInt64Values(s string) []int64 {
	strs := strings.Split(s, ",")
	var r []int64
	for _, str := range strs {
		i, err := strconv.ParseInt(str, 0, 64)
		if err == nil {
			r = append(r, i)
		}
	}
	return r
}
