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

// Package packages provides package management functions for Windows and Linux
// systems.
package packages

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/GoogleCloudPlatform/compute-image-tools/osinfo"
)

var (
	// AptExists indicates whether apt is installed.
	AptExists bool
	// YumExists indicates whether yum is installed.
	YumExists bool
	// ZypperExists indicates whether zypper is installed.
	ZypperExists bool
	// GemExists indicates whether gem is installed.
	GemExists bool
	// PipExists indicates whether pip is installed.
	PipExists bool
	// GooGetExists indicates whether googet is installed.
	GooGetExists bool

	noarch = osinfo.Architecture("noarch")
)

// PkgInfo describes a package.
type PkgInfo struct {
	Name, Arch, Version string
}

func run(cmd *exec.Cmd) ([]byte, error) {
	fmt.Printf("Running %q with args %q\n", cmd.Path, cmd.Args[1:])
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running %q with args %q: %v, stdout: %s", cmd.Path, cmd.Args, err, out)
	}
	return out, nil
}

func exists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}
