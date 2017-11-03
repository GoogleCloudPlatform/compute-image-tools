/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package packages

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/google/logger"
)

var (
	// dpkg-query
	dpkgquery     = "/usr/bin/dpkg-query"
	dpkgqueryArgs = []string{"-W", "-f", `${Package} ${Architecture} ${Version}\n`}

	// apt-get
	aptGet               = "/usr/bin/apt-get"
	aptGetUpdateArgs     = []string{"update"}
	aptGetUpgradableArgs = []string{"upgrade", "--just-print"}

	// rpmquery
	rpmquery     = "/usr/bin/rpmquery"
	rpmqueryArgs = []string{"-a", "--queryformat", `%{NAME} %{ARCH} %{VERSION}-%{RELEASE}\n`}

	// yum
	yum                 = "/usr/bin/yum"
	yumCheckUpdatesArgs = []string{"check-updates", "--quiet"}

	// zypper
	zypper                = "/usr/bin/zypper"
	zypperListUpdatesArgs = []string{"-q", "list-updates"}

	// gem
	gem             = "/usr/bin/gem"
	gemListArgs     = []string{"list", "--local"}
	gemOutdatedArgs = []string{"outdated", "--local"}

	// pip
	pip             = "/usr/bin/pip"
	pipListArgs     = []string{"list", "--format=legacy"}
	pipOutdatedArgs = append(pipListArgs, "--outdated")

	noarch = architecture("noarch")
)

// GetPackageUpdates gets all available package updates from any known
// installed package manager.
func GetPackageUpdates() (map[string][]PkgInfo, []string) {
	pkgs := map[string][]PkgInfo{}
	var errs []string
	if exists(aptGet) {
		apt, err := aptUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting apt updates: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["apt"] = apt
		}
	}
	if exists(yum) {
		yum, err := yumUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting yum updates: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["yum"] = yum
		}
	}
	if exists(zypper) {
		zypper, err := zypperUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting zypper updates: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["zypper"] = zypper
		}
	}
	if exists(gem) {
		gem, err := gemUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting gem updates: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["gem"] = gem
		}
	}
	if exists(pip) {
		pip, err := pipUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting pip updates: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["pip"] = pip
		}
	}
	return pkgs, errs
}

func aptUpdates() ([]PkgInfo, error) {
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
	   Inst libisc-export160 [1:9.10.3.dfsg.P4-12.3+deb9u2] (1:9.10.3.dfsg.P4-12.3+deb9u3 Debian:stable-updates [amd64])
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
		if len(pkg) != 6 {
			logger.Errorf("%q does not represent an apt update", ln)
			continue
		}
		ver := strings.Trim(pkg[3], "(")
		arch := strings.Trim(pkg[5], "[])")
		pkgs = append(pkgs, PkgInfo{Name: pkg[1], Arch: architecture(arch), Version: ver})
	}
	return pkgs, nil
}

func yumUpdates() ([]PkgInfo, error) {
	logger.Infof("Running %q with args %q", yum, yumCheckUpdatesArgs)
	out, err := exec.Command(yum, yumCheckUpdatesArgs...).CombinedOutput()
	// Exit code 0 means no updates, 100 means there are updates.
	if err == nil {
		return nil, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.ExitStatus() == 100 {
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error checking yum upgradable packages: %v, stdout: %s", err, out)
	}
	/*

	   foo.noarch 2.0.0-1 repo
	   bar.x86_64 2.0.0-1 repo
	   ...
	   Obsoleting Packages
	   ...
	*/

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[1:] {
		pkg := strings.Fields(ln)
		if pkg[0] == "Obsoleting" && pkg[1] == "Packages" {
			break
		}
		if len(pkg) != 3 {
			logger.Errorf("%s does not represent a yum update", ln)
			continue
		}
		name := strings.Split(pkg[0], ".")
		pkgs = append(pkgs, PkgInfo{Name: name[0], Arch: architecture(name[1]), Version: pkg[1]})
	}
	return pkgs, nil
}

func zypperUpdates() ([]PkgInfo, error) {
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
			logger.Errorf("%s does not represent a zypper update", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[4], Arch: architecture(pkg[10]), Version: pkg[8]})
	}
	return pkgs, nil
}

func gemUpdates() ([]PkgInfo, error) {
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
			logger.Errorf("%q does not represent a gem update", ln)
			continue
		}
		ver := strings.Trim(pkg[3], ")")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}

func pipUpdates() ([]PkgInfo, error) {
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
			logger.Errorf("%q does not represent a pip update", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: pkg[4]})
	}
	return pkgs, nil
}

// GetInstalledPackages gets all installed packages from any known installed
// package manager.
func GetInstalledPackages() (map[string][]PkgInfo, []string) {
	pkgs := map[string][]PkgInfo{}
	var errs []string
	if exists(rpmquery) {
		rpm, err := installedRPM()
		if err != nil {
			msg := fmt.Sprintf("error listing installed rpm packages: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["rpm"] = rpm
		}
	}
	if exists(dpkgquery) {
		deb, err := installedDEB()
		if err != nil {
			msg := fmt.Sprintf("error listing installed deb packages: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["deb"] = deb
		}
	}
	if exists(gem) {
		gem, err := installedGEM()
		if err != nil {
			msg := fmt.Sprintf("error listing installed gem packages: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["gem"] = gem
		}
	}
	if exists(pip) {
		pip, err := installedPIP()
		if err != nil {
			msg := fmt.Sprintf("error listing installed pip packages: %v", err)
			logger.Error(msg)
			errs = append(errs, msg)
		} else {
			pkgs["pip"] = pip
		}
	}
	return pkgs, errs
}

func installedDEB() ([]PkgInfo, error) {
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
		logger.Info("No deb packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			logger.Errorf("%q does not represent a deb", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}

func installedRPM() ([]PkgInfo, error) {
	out, err := run(exec.Command(rpmquery, rpmqueryArgs...))
	if err != nil {
		return nil, err
	}

	/*
	   foo x86_64 1.2.3-4
	   bar noarch 1.2.3-4
	   ...
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) == 0 {
		logger.Info("No rpm packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			logger.Errorf("%q does not represent a rpm", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}

func installedGEM() ([]PkgInfo, error) {
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
		logger.Info("No gems installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) < 2 {
			logger.Errorf("%q does not represent a gem", ln)
			continue
		}
		for _, ver := range strings.Split(strings.Trim(pkg[1], "()"), ", ") {
			pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
		}
	}
	return pkgs, nil
}

func installedPIP() ([]PkgInfo, error) {
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
		logger.Info("No python packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 2 {
			logger.Errorf("%q does not represent a python packages", ln)
			continue
		}
		ver := strings.Trim(pkg[1], "()")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}
