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

	"github.com/GoogleCloudPlatform/osconfig/packages"
)

const sha2Windows2008R2KB = "KB4474419"

type sha2DriverSigningCheck struct{}

func (s *sha2DriverSigningCheck) getName() string {
	return "SHA2 Driver Signing Check"
}

func (s *sha2DriverSigningCheck) run() (*report, error) {
	r := &report{name: s.getName()}
	if runtime.GOOS != "windows" || !strings.Contains(osInfo.Version, "6.1") {
		r.skipped = true
		r.Info("Only applicable on Windows 2008 systems.")
		return r, nil
	}

	pkgs, err := packages.GetInstalledPackages()
	if err != nil {
		return nil, fmt.Errorf("GetInstalledPackages error: %s", err)
	}

	for _, pkg := range pkgs.WUA {
		for _, id := range pkg.KBArticleIDs {
			if id == sha2Windows2008R2KB {
				r.Info(fmt.Sprintf("Windows Update containing SHA2 driver signing support found: %v", pkg))
				return r, nil
			}
		}
	}
	for _, pkg := range pkgs.QFE {
		if pkg.HotFixID == sha2Windows2008R2KB {
			r.Info(fmt.Sprintf("Windows Update containing SHA2 driver signing support found: %v", pkg))
			return r, nil
		}
	}
	r.Fatal(fmt.Sprintf("%s is required to support SHA2-signed drivers.", sha2Windows2008R2KB))
	return r, nil
}
