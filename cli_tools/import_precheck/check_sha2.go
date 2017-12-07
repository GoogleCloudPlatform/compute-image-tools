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
package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/packages"
)

const SHA2WINDOWS2008R2KB = "KB3033929"
const WINDOWS2008R2ROLLUPKB = "KB3125574"

type SHA2DriverSigningCheck struct{}

func (s *SHA2DriverSigningCheck) GetName() string {
	return "SHA2 Driver Signing Check"
}

func (s *SHA2DriverSigningCheck) Run() (*Report, error) {
	r := &Report{Name: s.GetName()}
	if runtime.GOOS != "windows" || !strings.Contains(osInfo.Version, "6.1") {
		r.Skipped = true
		r.Info("Only applicable on Windows 2008 systems.")
		return r, nil
	}

	pkgs, errs := packages.GetInstalledPackages()
	if errs != nil {
		return nil, fmt.Errorf("GetInstalledPackages errors:\n* %s", strings.Join(errs, "\n* "))
	}

	for _, pkg := range append(pkgs["qfe"], pkgs["wua"]...) {
		if pkg.Version == SHA2WINDOWS2008R2KB || pkg.Version == WINDOWS2008R2ROLLUPKB {
			r.Info(fmt.Sprintf("Windows Update containing SHA2 driver signing support found: %v", pkg))
			return r, nil
		}
	}
	r.Fatal("SHA2 driver signing support not found.")
	return r, nil
}
