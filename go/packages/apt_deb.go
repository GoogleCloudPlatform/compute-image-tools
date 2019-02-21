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
	// dpkg-query
	dpkgquery string
	aptGet    string

	dpkgqueryArgs        = []string{"-W", "-f", `${Package} ${Architecture} ${Version}\n`}
	aptGetInstallArgs    = []string{"install", "-y"}
	aptGetRemoveArgs     = []string{"remove", "-y"}
	aptGetUpdateArgs     = []string{"update"}
	aptGetUpgradeArgs    = []string{"upgrade", "-y"}
	aptGetUpgradableArgs = []string{"upgrade", "--just-print"}
)

func init() {
	if runtime.GOOS != "windows" {
		dpkgquery = "/usr/bin/dpkg-query"
		aptGet = "/usr/bin/apt-get"
	}
}

// InstallAptPackages installs apt packages.
func InstallAptPackages(pkgs []string) error {
	args := append(aptGetInstallArgs, pkgs...)
	out, err := run(exec.Command(aptGet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf(" %s\n", s)
	}
	DebugLogger.Printf("apt install output:\n%s\n", msg)
	return nil
}

// RemoveAptPackages removes apt packages.
func RemoveAptPackages(pkgs []string) error {
	args := append(aptGetRemoveArgs, pkgs...)
	out, err := run(exec.Command(aptGet, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf(" %s\n", s)
	}
	DebugLogger.Printf("apt remove output:\n%s\n", msg)
	return nil
}

// update apt packages
func aptUpgrade(run runFunc) error {
	if _, err := run(exec.Command(aptGet, aptGetUpdateArgs...)); err != nil {
		return err
	}

	if _, err := run(exec.Command(aptGet, aptGetUpgradeArgs...)); err != nil {
		return err
	}

	return nil
}

func aptUpdates(run runFunc) ([]PkgInfo, error) {
	out, err := run(exec.Command(aptGet, aptGetUpdateArgs...))
	if err != nil {
		return nil, err
	}

	out, err = run(exec.Command(aptGet, aptGetUpgradableArgs...))
	if err != nil {
		return nil, err
	}
	/*
	   NOTE: This is only a simulation!
	         apt-get needs root privileges for real execution.
	         Keep also in mind that locking is deactivated,
	         so don't depend on the relevance to the real current situation!
	   Reading package lists... Done
	   Building dependency tree
	   Reading state information... Done
	   Calculating upgrade... Done
	   The following packages will be upgraded:
	     google-cloud-sdk libdns-export162 libisc-export160
	   3 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.
	   Inst google-cloud-sdk [168.0.0-0] (171.0.0-0 cloud-sdk-stretch:cloud-sdk-stretch [all])
	   Inst python2.7 [2.7.13-2] (2.7.13-2+deb9u2 Debian:9.3/stable [amd64]) []
	   Inst libdns-export162 [1:9.10.3.dfsg.P4-12.3+deb9u2] (1:9.10.3.dfsg.P4-12.3+deb9u3 Debian:stable-updates [amd64])
	   Conf google-cloud-sdk (171.0.0-0 cloud-sdk-stretch:cloud-sdk-stretch [all])
	   Conf libisc-export160 (1:9.10.3.dfsg.P4-12.3+deb9u3 Debian:stable-updates [amd64])
	   Conf libdns-export162 (1:9.10.3.dfsg.P4-12.
	*/

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if pkg[0] != "Inst" {
			continue
		}
		if len(pkg) < 6 {
			DebugLogger.Printf("%q does not represent an apt update\n", ln)
			continue
		}
		ver := strings.Trim(pkg[3], "(")
		arch := strings.Trim(pkg[5], "[])")
		pkgs = append(pkgs, PkgInfo{Name: pkg[1], Arch: osinfo.Architecture(arch), Version: ver})
	}
	return pkgs, nil
}

func installedDEB(run runFunc) ([]PkgInfo, error) {
	out, err := run(exec.Command(dpkgquery, dpkgqueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   foo amd64 1.2.3-4
	   bar noarch 1.2.3-4
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) == 0 {
		DebugLogger.Println("No deb packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			DebugLogger.Printf("%q does not represent a deb\n", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: osinfo.Architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}
