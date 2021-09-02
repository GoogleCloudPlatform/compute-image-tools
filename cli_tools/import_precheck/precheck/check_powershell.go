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
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

var powershellVersions = []byte{'3', '4', '5'}

// PowershellCheck is a precheck.Check that verifies Powershell is installed,
// and that its version is sufficient to perform translation.
type PowershellCheck struct{}

// GetName returns the name of the precheck step; this is shown to the user.
func (c *PowershellCheck) GetName() string {
	return "Powershell Check"
}

// Run executes the precheck step.
func (c *PowershellCheck) Run() (*Report, error) {
	r := &Report{name: c.GetName()}
	if runtime.GOOS != "windows" {
		r.skipped = true
		r.Info("Not applicable on non-Windows systems.")
		return r, nil
	}

	out, err := exec.Command("powershell", "-Command", "$PSVersionTable.PSVersion.Major").Output()
	if err != nil {
		return nil, err
	}
	out = out[:1]
	var found bool
	if bytes.Contains(powershellVersions, out) {
		found = true
	}
	r.Info(fmt.Sprintf("Powershell %s detected.", string(out)))
	if !found {
		r.Warn("Powershell 3+ recommended.")
	}
	return r, nil
}
