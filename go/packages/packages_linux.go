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
	"strings"
)

// UpdatePackages installs all available package updates for all known system
// package managers.
func UpdatePackages() error {
	var errs []string
	if AptExists {
		if err := aptUpgrade(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if YumExists {
		if err := yumUpdate(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if ZypperExists {
		if err := zypperUpdate(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if errs == nil {
		return nil
	}
	return errors.New(strings.Join(errs, ",\n"))
}

// GetPackageUpdates gets all available package updates from any known
// installed package manager.
func GetPackageUpdates() (Packages, []string) {
	pkgs := Packages{}
	var errs []string
	if AptExists {
		apt, err := AptUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting apt updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Apt = apt
		}
	}
	if YumExists {
		yum, err := YumUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting yum updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Yum = yum
		}
	}
	if ZypperExists {
		zypper, err := ZypperUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting zypper updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Zypper = zypper
		}
	}
	if GemExists {
		gem, err := GemUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting gem updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Gem = gem
		}
	}
	if PipExists {
		pip, err := PipUpdates()
		if err != nil {
			msg := fmt.Sprintf("error getting pip updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Pip = pip
		}
	}
	return pkgs, errs
}

// GetInstalledPackages gets all installed packages from any known installed
// package manager.
func GetInstalledPackages() (Packages, []string) {
	pkgs := Packages{}
	var errs []string
	if exists(rpmquery) {
		rpm, err := InstalledRPMPackages()
		if err != nil {
			msg := fmt.Sprintf("error listing installed rpm packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Rpm = rpm
		}
	}
	if exists(dpkgquery) {
		deb, err := InstalledDebPackages()
		if err != nil {
			msg := fmt.Sprintf("error listing installed deb packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Deb = deb
		}
	}
	if exists(gem) {
		gem, err := InstalledGemPackages()
		if err != nil {
			msg := fmt.Sprintf("error listing installed gem packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Gem = gem
		}
	}
	if exists(pip) {
		pip, err := InstalledPipPackages()
		if err != nil {
			msg := fmt.Sprintf("error listing installed pip packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.Pip = pip
		}
	}
	return pkgs, errs
}
