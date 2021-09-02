//  Copyright 2017 Google Inc. All Rights Reserved.
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

package precheck

import (
	"fmt"
	"strings"
)

// Result indicates the result of the check.
type Result int

const (
	// Passed means the check passed.
	Passed Result = iota

	// Failed means the check ran and found an error such that
	// the machine is not importable.
	Failed

	// Skipped means the check didn't run.
	Skipped

	// Unknown means the test couldn't run, and that the user
	// should run a manual verification.
	Unknown
)

func (r Result) String() string {
	switch r {
	case Passed:
		return "PASSED"
	case Failed:
		return "FAILED"
	case Skipped:
		return "SKIPPED"
	case Unknown:
		return "UNKNOWN"
	default:
		panic(fmt.Sprintf("Unmapped status: %d", r))
	}
}

// Report contains the results of running one or more precheck steps.
type Report struct {
	name   string
	result Result
	logs   []string
}

// Failed returns whether one or more precheck steps failed.
func (r *Report) Failed() bool {
	return r.result == Failed
}

// Fatal messages indicate that the system is not importable.
func (r *Report) Fatal(s string) {
	r.result = Failed
	r.logs = append(r.logs, "FATAL: "+s)
}

// Info messages are informational and shown to the user.
func (r *Report) Info(s string) {
	r.logs = append(r.logs, "INFO: "+s)
}

// Warn messages indicate that the user should perform a manual check.
func (r *Report) Warn(s string) {
	r.logs = append(r.logs, "WARN: "+s)
}

func (r *Report) String() string {
	title := strings.Join([]string{r.name, r.result.String()}, " -- ")
	border := strings.Repeat("#", len(title)+4)

	lines := []string{border, "# " + title + " #", border}
	for _, l := range r.logs {
		lines = append(lines, "  * "+l)
	}
	return strings.Join(lines, "\n")
}
