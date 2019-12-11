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
)

// GetPackageUpdates gets available package updates GooGet as well as any
// available updates from Windows Update Agent.
func GetPackageUpdates() (Packages, []string) {
	var pkgs Packages
	var errs []string

	if GooGetExists {
		if googet, err := GooGetUpdates(); err != nil {
			msg := fmt.Sprintf("error listing googet updates: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.GooGet = googet
		}
	}
	if wua, err := WUAUpdates("IsInstalled=0"); err != nil {
		msg := fmt.Sprintf("error listing installed Windows updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.WUA = wua
	}
	return pkgs, errs
}

// GetInstalledPackages gets all installed GooGet packages and Windows updates.
// Windows updates are read from Windows Update Agent and Win32_QuickFixEngineering.
func GetInstalledPackages() (Packages, []string) {
	var pkgs Packages
	var errs []string

	if exists(googet) {
		if googet, err := InstalledGooGetPackages(); err != nil {
			msg := fmt.Sprintf("error listing installed googet packages: %v", err)
			DebugLogger.Println("Error:", msg)
			errs = append(errs, msg)
		} else {
			pkgs.GooGet = googet
		}
	}

	if wua, err := WUAUpdates("IsInstalled=1"); err != nil {
		msg := fmt.Sprintf("error listing installed Windows updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.WUA = wua
	}

	if qfe, err := QuickFixEngineering(); err != nil {
		msg := fmt.Sprintf("error listing installed QuickFixEngineering updates: %v", err)
		DebugLogger.Println("Error:", msg)
		errs = append(errs, msg)
	} else {
		pkgs.QFE = qfe
	}

	return pkgs, errs
}
