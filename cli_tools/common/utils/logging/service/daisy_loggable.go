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
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

// NewLoggableFromWorkflow provides a Loggable from a daisy workflow.
func NewLoggableFromWorkflow(wf *daisy.Workflow) Loggable {
	if wf == nil {
		return nil
	}
	return workflowLoggable{wf: wf}
}

type workflowLoggable struct {
	wf *daisy.Workflow
}

func (w workflowLoggable) GetValue(key string) string {
	return w.wf.GetSerialConsoleOutputValue(key)
}

func (w workflowLoggable) GetValueAsBool(key string) bool {
	v, err := strconv.ParseBool(w.wf.GetSerialConsoleOutputValue(key))
	if err != nil {
		return false
	}
	return v
}

func (w workflowLoggable) GetValueAsInt64Slice(key string) []int64 {
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
