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
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
)

var (
	// dpkg-query
	dpkgquery     = "/usr/bin/dpkg-query"
	dpkgqueryArgs = []string{"-W", "-f", `${Package} ${Architecture} ${Version}\n`}

	// apt-get
	aptGet               = "/usr/bin/apt-get"
	aptGetUpdateArgs     = []string{"update"}
	aptGetUpgradeArgs    = []string{"upgrade", "-y"}
	aptGetUpgradableArgs = []string{"upgrade", "--just-print"}

	// rpmquery
	rpmquery     = "/usr/bin/rpmquery"
	rpmqueryArgs = []string{"-a", "--queryformat", `%{NAME} %{ARCH} %{VERSION}-%{RELEASE}\n`}

	// yum
	yum                 = "/usr/bin/yum"
	yumUpdateArgs       = []string{"update", "-y"}
	yumCheckUpdatesArgs = []string{"check-updates", "--quiet"}

	// zypper
	zypper                = "/usr/bin/zypper"
	zypperInstallArgs     = []string{"install", "--no-confirm"}
	zypperRemoveArgs      = []string{"remove", "--no-confirm"}
	zypperUpdateArgs      = []string{"update"}
	zypperListArgs        = []string{"packages", "--installed-only"}
	zypperListUpdatesArgs = []string{"-q", "list-updates"}

	// gem
	gem             = "/usr/bin/gem"
	gemListArgs     = []string{"list", "--local"}
	gemOutdatedArgs = []string{"outdated", "--local"}

	// pip
	pip             = "/usr/bin/pip"
	pipListArgs     = []string{"list", "--format=legacy"}
	pipOutdatedArgs = append(pipListArgs, "--outdated")
)

func init() {
	AptExists = exists(aptGet)
	YumExists = exists(yum)
	ZypperExists = exists(zypper)
	GemExists = exists(gem)
	PipExists = exists(pip)
}

// InstallAptPackages installs apt packages.
func InstallAptPackages(pkgs []string) {}

// RemoveAptPackages removes apt packages.
func RemoveAptPackages(pkgs []string) {}

// InstallYumPackages installs yum packages.
func InstallYumPackages(pkgs []string) {}

// RemoveYumPackages removes yum packages.
func RemoveYumPackages(pkgs []string) {}

// Installs zypper packages
func InstallZypperPackages(Run RunFunc, pkgs []string) error {
	args := append(zypperInstallArgs, pkgs...)
	out, err := Run(exec.Command(zypper, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf(" %s\n", s)
	}
	fmt.Printf("Zypper install output:\n%s\n", msg)
	return nil
}

// RemoveZypperPackages installed Zypper packages.
func RemoveZypperPackages(Run RunFunc, pkgs []string) error {
	args := append(zypperRemoveArgs, pkgs...)
	out, err := Run(exec.Command(zypper, args...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	fmt.Printf("Zypper remove output:\n%s\n", msg)
	return nil
}

// InstallZypperUpdates installs all available Zypper updates.
func InstallZypperUpdates(Run RunFunc) error {
	out, err := Run(exec.Command(zypper, zypperUpdateArgs...))
	if err != nil {
		return err
	}
	var msg string
	for _, s := range strings.Split(string(out), "\n") {
		msg += fmt.Sprintf("  %s\n", s)
	}
	fmt.Printf("Zypper update output:\n%s\n", msg)
	return nil
}

// UpdatePackages installs all available package updates for all known system
// package managers.
func UpdatePackages(Run RunFunc) error {
	var errs []string
	if AptExists {
		if err := aptUpgrade(Run); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if YumExists {
		if err := yumUpdate(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if ZypperExists {
		if err := zypperUpdate(Run); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

// update apt packages
func aptUpgrade(Run RunFunc) error {
	if _, err := Run(exec.Command(aptGet, aptGetUpdateArgs...)); err != nil {
		return err
	}

	if _, err := Run(exec.Command(aptGet, aptGetUpgradeArgs...)); err != nil {
		return err
	}

	return nil
}

// update yum packages
func yumUpdate() error {
	if _, err := exec.Command(yum, yumUpdateArgs...).CombinedOutput(); err != nil {
		return err
	}
	return nil
}

// update zypper packages
func zypperUpdate(Run RunFunc) error {
	if _, err := Run(exec.Command(zypper, zypperUpdateArgs...)); err != nil {
		return err
	}
	return nil
}

// GetPackageUpdates gets all available package updates from any known
// installed package manager.
func GetPackageUpdates(Run RunFunc) (Packages, []string) {
	pkgs := Packages{}
	var errs []string
	if AptExists {
		apt, err := aptUpdates(Run)
		if err != nil {
			msg := fmt.Sprintf("error getting apt updates: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Apt = apt
		}
	}
	if YumExists {
		yum, err := yumUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting yum updates: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Yum = yum
		}
	}
	if ZypperExists {
		zypper, err := zypperUpdates(Run)
		if err != nil {
			msg := fmt.Sprintf("error getting zypper updates: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Zypper = zypper
		}
	}
	if GemExists {
		gem, err := gemUpdates(Run)
		if err != nil {
			msg := fmt.Sprintf("error getting gem updates: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Gem = gem
		}
	}
	if PipExists {
		pip, err := pipUpdates(Run)
		if err != nil {
			msg := fmt.Sprintf("error getting pip updates: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Pip = pip
		}
	}
	return pkgs, errs
}

func aptUpdates(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(aptGet, aptGetUpdateArgs...))
	if err != nil {
		return nil, err
	}

	out, err = Run(exec.Command(aptGet, aptGetUpgradableArgs...))
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
			fmt.Printf("%q does not represent an apt update\n", ln)
			continue
		}
		ver := strings.Trim(pkg[3], "(")
		arch := strings.Trim(pkg[5], "[])")
		pkgs = append(pkgs, PkgInfo{Name: pkg[1], Arch: osinfo.Architecture(arch), Version: ver})
	}
	return pkgs, nil
}

func yumUpdates() ([]PkgInfo, error) {
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
			fmt.Printf("%s does not represent a yum update\n", ln)
			continue
		}
		name := strings.Split(pkg[0], ".")
		pkgs = append(pkgs, PkgInfo{Name: name[0], Arch: osinfo.Architecture(name[1]), Version: pkg[1]})
	}
	return pkgs, nil
}

func zypperUpdates(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(zypper, zypperListUpdatesArgs...))
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
			fmt.Printf("%s does not represent a zypper update\n", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[4], Arch: osinfo.Architecture(pkg[10]), Version: pkg[8]})
	}
	return pkgs, nil
}

func gemUpdates(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(gem, gemOutdatedArgs...))
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
			fmt.Printf("%q does not represent a gem update\n", ln)
			continue
		}
		ver := strings.Trim(pkg[3], ")")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}

func pipUpdates(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(pip, pipOutdatedArgs...))
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
			fmt.Printf("%q does not represent a pip update\n", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: pkg[4]})
	}
	return pkgs, nil
}

// GetInstalledPackages gets all installed packages from any known installed
// package manager.
func GetInstalledPackages(Run RunFunc) (Packages, []string) {
	pkgs := Packages{}
	var errs []string
	if exists(rpmquery) {
		rpm, err := installedRPM(Run)
		if err != nil {
			msg := fmt.Sprintf("error listing installed rpm packages: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Rpm = rpm
		}
	}
	if exists(dpkgquery) {
		deb, err := installedDEB(Run)
		if err != nil {
			msg := fmt.Sprintf("error listing installed deb packages: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Deb = deb
		}
	}
	if exists(gem) {
		gem, err := installedGEM(Run)
		if err != nil {
			msg := fmt.Sprintf("error listing installed gem packages: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Gem = gem
		}
	}
	if exists(zypper) {
		zypper, err := installedZypper(Run)
		if err != nil {
			msg := fmt.Sprintf("error listing installed zypper packages: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Zypper = zypper
		}
	}

	if exists(pip) {
		pip, err := installedPIP(Run)
		if err != nil {
			msg := fmt.Sprintf("error listing installed pip packages: %v", err)
			fmt.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Pip = pip
		}
	}
	return pkgs, errs
}

func installedDEB(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(dpkgquery, dpkgqueryArgs...))
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
		fmt.Println("No deb packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			fmt.Printf("%q does not represent a deb\n", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: osinfo.Architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}

func installedRPM(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(rpmquery, rpmqueryArgs...))
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
		fmt.Println("No rpm packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			fmt.Printf("%q does not represent a rpm\n", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: osinfo.Architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}

func installedGEM(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(gem, gemListArgs...))
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
		fmt.Println("No gems installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) < 2 {
			fmt.Printf("%q does not represent a gem\n", ln)
			continue
		}
		for _, ver := range strings.Split(strings.Trim(pkg[1], "()"), ", ") {
			pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
		}
	}
	return pkgs, nil
}

func installedPIP(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(pip, pipListArgs...))
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
			fmt.Printf("%q does not represent a python packages\n", ln)
			continue
		}
		ver := strings.Trim(pkg[1], "()")
		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: noarch, Version: ver})
	}
	return pkgs, nil
}

func installedZypper(Run RunFunc) ([]PkgInfo, error) {
	out, err := Run(exec.Command(zypper, zypperListArgs...))
	if err != nil {
		return nil, err
	}

	/*
		S  | Repository                             | Name                                    | Version                             | Arch
		---+----------------------------------------+-----------------------------------------+-------------------------------------+-------
		i  | SLE-Module-Basesystem15-Pool           | GeoIP-data                              | 1.6.11-1.19                         | noarch
		v  | SLE-Module-Basesystem15-Updates        | SUSEConnect                             | 0.3.14-3.13.1                       | x86_64
		v  | SLE-Module-Basesystem15-Updates        | SUSEConnect                             | 0.3.12-3.10.1                       | x86_64
		i+ | SLE-Module-Basesystem15-Updates        | SUSEConnect                             | 0.3.11-3.3.1                        | x86_64
		v  | SLE-Module-Basesystem15-Pool           | SUSEConnect                             | 0.3.11-1.4                          | x86_64
		i+ | @System                                | SuSEfirewall2                           | 3.6.378-1.33                        | noarch
	*/
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	if len(lines) == 0 {
		fmt.Println("No zypper packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines[2:] {
		pkg := strings.Fields(ln)
		if len(pkg) != 9 {
			fmt.Printf("%q does not represent a zypper packages\n", ln)
			continue
		}
		pkgs = append(pkgs, PkgInfo{Name: pkg[4], Arch: osinfo.Architecture(pkg[8]), Version: pkg[6]})
	}
	return pkgs, nil
}
