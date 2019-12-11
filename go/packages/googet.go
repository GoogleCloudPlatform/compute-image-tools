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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	googet string

	googetUpdateArgs         = []string{"-noconfirm", "update"}
	googetUpdateQueryArgs    = []string{"update"}
	googetInstalledQueryArgs = []string{"installed"}
	googetInstallArgs        = []string{"-noconfirm", "install"}
	googetRemoveArgs         = []string{"-noconfirm", "remove"}
)

func init() {
	if runtime.GOOS == "windows" {
		googet = filepath.Join(os.Getenv("GooGetRoot"), "googet.exe")
	}
	GooGetExists = exists(googet)
}

// GooGetUpdates queries for all available googet updates.
func GooGetUpdates() ([]PkgInfo, error) {
	out, err := run(exec.Command(googet, googetUpdateQueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   Searching for available updates...
	   foo.noarch, 3.5.4@1 --> 3.6.7@1 from repo
	   ...
	   Perform update? (y/N):
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var pkgs []PkgInfo
	for _, ln := range lines[1:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 6 {
			continue
		}

		p := strings.Split(pkg[0], ".")
		if len(p) != 2 {
			DebugLogger.Printf("%s does not represent a package", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: p[0], Arch: strings.Trim(p[1], ","), Version: pkg[3]})
	}
	return pkgs, nil
}

// InstallGooGetPackages installs GooGet packages.
func InstallGooGetPackages(pkgs []string) error {
	args := append(googetInstallArgs, pkgs...)
	out, err := run(exec.Command(googet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet install output:\n%s", msg)
	return nil
}

// RemoveGooGetPackages installs GooGet packages.
func RemoveGooGetPackages(pkgs []string) error {
	args := append(googetRemoveArgs, pkgs...)
	out, err := run(exec.Command(googet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet remove output:\n%s", msg)
	return nil
}

// InstallGooGetUpdates installs all available GooGet updates.
func InstallGooGetUpdates() error {
	out, err := run(exec.Command(googet, googetUpdateArgs...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	DebugLogger.Printf("GooGet update output:\n%s", msg)
	return nil
}

// InstalledGooGetPackages queries for all installed googet packages.
func InstalledGooGetPackages() ([]PkgInfo, error) {
	out, err := run(exec.Command(googet, googetInstalledQueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   Installed Packages:
	   foo.x86_64 1.2.3@4
	   bar.noarch 1.2.3@4
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) <= 1 {
		DebugLogger.Println("No packages GooGet installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[1:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 2 {
			DebugLogger.Printf("%s does not represent a GooGet package", ln)
			continue
		}

		p := strings.Split(pkg[0], ".")
		if len(p) != 2 {
			DebugLogger.Printf("%s does not represent a GooGet package", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: p[0], Arch: p[1], Version: pkg[1]})
	}
	return pkgs, nil
}
