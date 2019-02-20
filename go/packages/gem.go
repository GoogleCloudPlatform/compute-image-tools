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
	"os/exec"
	"runtime"
	"strings"
)

var (
	gem string

	gemListArgs     = []string{"list", "--local"}
	gemOutdatedArgs = []string{"outdated", "--local"}
)

func init() {
	if runtime.GOOS != "windows" {
		gem = "/usr/bin/gem"
	}
}

func gemUpdates(run runFunc) ([]PkgInfo, error) {
	out, err := run(exec.Command(gem, gemOutdatedArgs...))
	if err != nil {
		return nil, err
	}
	/*
	   foo (1.2.8 < 1.3.2)
	   bar (1.0.0 < 1.1.2)
	   ...
	*/

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 4 {
			DebugLogger.Printf("%q does not represent a gem update\n", ln)
			continue
		}
		ver := strings.Trim(pkg[3], ")")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}

func installedGEM(run runFunc) ([]PkgInfo, error) {
	out, err := run(exec.Command(gem, gemListArgs...))
	if err != nil {
		return nil, err
	}

	/*

	   *** LOCAL GEMS ***

	   foo (1.2.3, 1.2.4)
	   bar (1.2.3)
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) == 0 {
		DebugLogger.Println("No gems installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) < 2 {
			DebugLogger.Printf("%q does not represent a gem\n", ln)
			continue
		}
		for _, ver := range strings.Split(strings.Trim(pkg[1], "()"), ", ") {
			pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
		}
	}
	return pkgs, nil
}
