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
	"bufio"
	"bytes"
	"fmt"
	"net"
	"runtime"
)

var sshVersions = [][]byte{
	[]byte("OpenSSH"),
}

// SSHCheck is a precheck.Check that confirms that port 22 is open and that
// the SSH server daemon is running.
type SSHCheck struct{}

// GetName returns the name of the precheck step; this is shown to the user.
func (c *SSHCheck) GetName() string {
	return "SSH Check"
}

// Run executes the precheck step.
func (c *SSHCheck) Run() (*Report, error) {
	r := &Report{name: c.GetName()}
	if runtime.GOOS == "windows" {
		r.skipped = true
		r.Info("Not applicable on Windows systems.")
		return r, nil
	}

	conn, err := net.Dial("tcp", "localhost:22")
	if err != nil {
		r.Warn("port 22 closed, gcloud and Cloud Console SSH clients will not work.")
	}

	data := make([]byte, 512)
	_, err = bufio.NewReader(conn).Read(data)
	if err != nil {
		return nil, err
	}

	var found []byte
	for _, version := range sshVersions {
		if bytes.Contains(data, version) {
			found = version
			break
		}
	}
	if found != nil {
		r.Info(fmt.Sprintf("SSH (%s) detected on port 22.", string(found)))
	} else {
		r.Warn("SSH not detected on port 22, gcloud and Cloud Console SSH clients will not work.")
	}

	return r, nil
}
