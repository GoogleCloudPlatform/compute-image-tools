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

// Package osinfo provides basic system info functions for Windows and
// Linux.
package osinfo

const (
	// Linux is the default shortname used for a Linux system.
	Linux = "linux"
	// Windows is the default shortname used for Windows system.
	Windows = "windows"
)

// DistributionInfo describes an OS distribution.
type DistributionInfo struct {
	LongName, ShortName, Version, Kernel, Architecture string
}

// Architecture attempts to standardize architecture naming.
func Architecture(arch string) string {
	switch arch {
	case "amd64", "64-bit":
		arch = "x86_64"
	case "i386", "i686", "32-bit":
		arch = "x86_32"
	case "noarch":
		arch = "all"
	}
	return arch
}
