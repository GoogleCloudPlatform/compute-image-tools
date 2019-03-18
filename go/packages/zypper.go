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

	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
)

var (
	zypper string

	zypperInstallArgs     = []string{"install", "--no-confirm"}
	zypperRemoveArgs      = []string{"remove", "--no-confirm"}
	zypperUpdateArgs      = []string{"update"}
	zypperListArgs        = []string{"packages", "--installed-only"}
	zypperListUpdatesArgs = []string{"-q", "list-updates"}
)

func init() {
	if runtime.GOOS != "windows" {
		zypper = "/usr/bin/zypper"
	}
	ZypperExists = exists(zypper)
}

// InstallZypperPackages Installs zypper packages
func InstallZypperPackages(pkgs []string) error {
	args := append(zypperInstallArgs, pkgs...)
	out, err := run(exec.Command(zypper, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf(" %s\n", s)
	}
	DebugLogger.Printf("Zypper install output:\n%s\n", msg)
	return nil
}

// RemoveZypperPackages installed Zypper packages.
func RemoveZypperPackages(pkgs []string) error {
	args := append(zypperRemoveArgs, pkgs...)
	out, err := run(exec.Command(zypper, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("Zypper remove output:\n%s\n", msg)
	return nil
}

// InstallZypperUpdates installs all available Zypper updates.
func InstallZypperUpdates() error {
	out, err := run(exec.Command(zypper, zypperUpdateArgs...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("Zypper update output:\n%s\n", msg)
	return nil
}

// update zypper packages
func zypperUpdate(run runFunc) error {
	if _, err := run(exec.Command(zypper, zypperUpdateArgs...)); err != nil {
		return err
	}
	return nil
}

func zypperUpdates(run runFunc) ([]PkgInfo, error) {
	out, err := run(exec.Command(zypper, zypperListUpdatesArgs...))
	if err != nil {
		return nil, err
	}
	/*
		      S | Repository          | Name                   | Current Version | Available Version | Arch
		      --+---------------------+------------------------+-----------------+-------------------+-------
		      v | SLES12-SP3-Updates  | at                     | 3.1.14-7.3      | 3.1.14-8.3.1      | x86_64
		      v | SLES12-SP3-Updates  | autoyast2-installation | 3.2.17-1.3      | 3.2.22-2.9.2      | noarch
			   ...
	*/

	// We could use the XML output option, but parsing the lines is inline
	// with other functions and pretty simple.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 11 {
			DebugLogger.Printf("%s does not represent a zypper update\n", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[4], Arch: osinfo.Architecture(pkg[10]), Version: pkg[8]})
	}
	return pkgs, nil
}
