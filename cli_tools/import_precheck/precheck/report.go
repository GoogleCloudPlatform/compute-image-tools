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
	"strings"
)

// Report contains the results of running one or more precheck steps.
type Report struct {
	name    string
	skipped bool
	failed  bool
	logs    []string
}

// Failed returns whether one or more precheck steps failed.
func (r *Report) Failed() bool {
	return r.failed
}

// Fatal messages indicate that the system is not importable.
func (r *Report) Fatal(s string) {
	r.failed = true
	r.logs = append(r.logs, "FATAL: "+s)
}

// Info messages are informational and shown to the user.
func (r *Report) Info(s string) {
	r.logs = append(r.logs, "INFO: "+s)
}

// Warn messages indicate that a precheck step couldn't run,
// and that the user should manually verify the check on their own.
func (r *Report) Warn(s string) {
	r.logs = append(r.logs, "WARN: "+s)
}

func (r *Report) String() string {
	status := "PASSED"
	if r.skipped {
		status = "SKIPPED"
	} else if r.failed {
		status = "FAILED"
	}

	title := strings.Join([]string{r.name, status}, " -- ")
	border := strings.Repeat("#", len(title)+4)

	lines := []string{border, "# " + title + " #", border}
	for _, l := range r.logs {
		lines = append(lines, "  * "+l)
	}
	return strings.Join(lines, "\n")
}
