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

package packages

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var (
	pip string

	pipListArgs     = []string{"list", "--format=legacy"}
	pipOutdatedArgs = append(pipListArgs, "--outdated")
)

func init() {
	if runtime.GOOS != "windows" {
		pip = "/usr/bin/pip"
	}
	PipExists = exists(pip)
}

// PipUpdates queries for all available pip updates.
func PipUpdates() ([]PkgInfo, error) {
	out, err := run(exec.Command(pip, pipOutdatedArgs...))
	if err != nil {
		return nil, err
	}
	/*
	   foo (4.5.3) - Latest: 4.6.0 [repo]
	   bar (1.3) - Latest: 1.4 [repo]
	   ...
	*/

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 6 {
			DebugLogger.Printf("%q does not represent a pip update\n", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: pkg[4]})
	}
	return pkgs, nil
}

// InstalledPipPackages queries for all installed pip packages.
func InstalledPipPackages() ([]PkgInfo, error) {
	out, err := run(exec.Command(pip, pipListArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   foo (1.2.3)
	   bar (1.2.3)
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) == 0 {
		fmt.Println("No python packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 2 {
			DebugLogger.Printf("%q does not represent a python packages\n", ln)
			continue
		}
		ver := strings.Trim(pkg[1], "()")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}
