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

	"github.com/GoogleCloudPlatform/compute-image-tools/go/osinfo"
)

var (
	rpmquery string

	rpmqueryArgs = []string{"-a", "--queryformat", `%{NAME} %{ARCH} %{VERSION}-%{RELEASE}\n`}
)

func init() {
	if runtime.GOOS != "windows" {
		rpmquery = "/usr/bin/rpmquery"
	}
}

func installedRPM(run runFunc) ([]PkgInfo, error) {
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
		DebugLogger.Println("No rpm packages installed.")
		return nil, nil
	}

	var pkgs []PkgInfo
	for _, ln := range lines {
		pkg := strings.Fields(ln)
		if len(pkg) != 3 {
			DebugLogger.Printf("%q does not represent a rpm\n", ln)
			continue
		}

		pkgs = append(pkgs, PkgInfo{Name: pkg[0], Arch: osinfo.Architecture(pkg[1]), Version: pkg[2]})
	}
	return pkgs, nil
}
