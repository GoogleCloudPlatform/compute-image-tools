//  Copyright 2021 Google Inc. All Rights Reserved.
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

package shell

import (
	"bufio"
	"bytes"
	"os/exec"
)

// To rebuild mocks, run `go generate ./...`
//go:generate go run github.com/golang/mock/mockgen -package mocks -source $GOFILE -mock_names=Executor=MockShellExecutor -destination ../../../mocks/mock_shell_exececutor.go

// Executor is a shim over cmd.Output() that allows for testing.
type Executor interface {
	// Exec executes program with args, and returns stdout if the return code is zero.
	// If nonzero, stderr is included in error.
	Exec(program string, args ...string) (string, error)
	// ExecLines is similar to Exec, except it splits the output on newlines. All empty
	// lines are discarded.
	ExecLines(program string, args ...string) ([]string, error)
}

// NewShellExecutor creates a shell.Executor that is implemented by exec.Command.
func NewShellExecutor() Executor {
	return &defaultShellExecutor{}
}

type defaultShellExecutor struct {
}

func (d *defaultShellExecutor) Exec(program string, args ...string) (string, error) {
	cmd := exec.Command(program, args...)
	stdout, err := cmd.Output()
	return string(stdout), err
}

func (d *defaultShellExecutor) ExecLines(program string, args ...string) (allLines []string, err error) {
	cmd := exec.Command(program, args...)
	stdout, err := cmd.Output()
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			allLines = append(allLines, line)
		}
	}
	return allLines, err
}
