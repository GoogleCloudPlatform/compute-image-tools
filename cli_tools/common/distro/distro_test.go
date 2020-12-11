//  Copyright 2020 Google Inc. All Rights Reserved.
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

package distro

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
)

func TestParseGcloudOsParam_HappyCases(t *testing.T) {
	daisy_utils.GetSortedOSIDs()
	for _, osID := range daisy_utils.GetSortedOSIDs() {
		t.Run(osID, func(t *testing.T) {
			d, e := FromGcloudOSArgument(osID)
			assert.NoError(t, e)
			var expected string
			if osID == "windows-8-1-x64-byol" {
				// windows-8-1-x64-byol is a legacy flag value, and it's the only value that
				// includes an extra hyphen between its major and minor version. The non-legacy
				// flag is windows-8-x64-byol.
				expected = "windows-8-x64"
			} else if strings.HasSuffix(osID, "-byol") {
				// The Release interface is orthogonal to license, so
				// its AsGcloudArg doesn't include license info.
				expected = osID[:len(osID)-5]
			} else {
				expected = osID
			}
			assert.Equal(t, expected, d.AsGcloudArg())
		})
	}
}

func TestParseGcloudOsParam_DistroNameErrors(t *testing.T) {
	var cases = []struct {
		in  string
		err string
	}{
		{"", "expected pattern of `distro-version`. Actual: ``"},
		{"unknown", "expected pattern of `distro-version`. Actual: `unknown`"},
		{"notSupported-18", "distro `notsupported` is not importable"},
		{"notSupported-1804", "distro `notsupported` is not importable"},
		{"sles", "expected pattern of `distro-version`. Actual: `sles`"},
		{"kali-12", "distro `kali` is not importable"},
		{"ubuntu", "expected pattern of `distro-version`"},
		{"ubuntu-12", "expected version with length four"},
		{"opensuse-15-leap", "expected pattern of `distro-version`. Actual: `opensuse-15-leap`"},
		{"debian", "expected pattern of `distro-version`. Actual: `debian`"},
		{"centos7", "expected pattern of `distro-version`. Actual: `centos7`"},
		{"rhel", "expected pattern of `distro-version`. Actual: `rhel`"},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			d, e := FromGcloudOSArgument(tt.in)
			assert.Nil(t, d)
			if e == nil {
				t.Fatalf("Expected error")
			}
			assert.Contains(t, e.Error(), tt.err)
		})
	}
}

func TestParseGcloudOsParam_VersionErrors(t *testing.T) {
	var cases = []struct {
		in  string
		err string
	}{
		{"ubuntu-aaaa", "major version required to be an integer greater than zero. Received: `aa`"},
		{"ubuntu-1", "expected version with length four. Actual: `1`"},
		{"ubuntu-11", "expected version with length four. Actual: `11`"},
		{"ubuntu-111", "expected version with length four. Actual: `111`"},
		{"ubuntu-1111", "Ubuntu version `11.11` is not importable"},
		{"centos-0", "major version required to be an integer greater than zero. Received: `0`"},
		{"centos-x", "major version required to be an integer greater than zero. Received: `x`"},
		{"sles-0", "major version required to be an integer greater than zero. Received: `0`"},
		{"sles-sap-", "expected pattern of `distro-version`. Actual: `sles-sap-`"},
		{"windows-vista", "`vista` is not a valid major version for Windows"},
		{"windows-x64", "`x64` is not a valid major version for Windows"},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			d, e := FromGcloudOSArgument(tt.in)
			assert.Nil(t, d)
			assert.Error(t, e)
			assert.Contains(t, e.Error(), tt.err)
		})
	}
}

func TestDistroFromComponents_HappyCasesLinux(t *testing.T) {
	var cases = []struct {
		distro, major, minor string
		expectedGcloud       string
	}{
		{"debian", "8", "", "debian-8"},
		{"debian", "8", "1", "debian-8"},
		{"centos", "7", "", "centos-7"},
		{"centos", "7", "5", "centos-7"},
		{"opensuse", "15", "", "opensuse-15"},
		{"opensuse", "15", "2", "opensuse-15"},
		{"sles", "12", "", "sles-12"},
		{"sles", "12", "1", "sles-12"},
		{"sles-sap", "12", "", "sles-sap-12"},
		{"rhel", "6", "", "rhel-6"},
		{"rhel", "8", "2", "rhel-8"},
		{"ubuntu", "14", "04", "ubuntu-1404"},
		{"ubuntu", "14", "10", "ubuntu-1410"},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s-%s-%s", tt.distro, tt.major, tt.minor), func(t *testing.T) {
			d, e := FromComponents(tt.distro, tt.major, tt.minor, "")
			assert.NoError(t, e)
			assert.Equal(t, tt.expectedGcloud, d.AsGcloudArg())
		})
	}
}

func TestDistroFromComponents_HappyCasesWindows(t *testing.T) {
	var cases = []struct {
		major, minor, arch string
		expectedGcloud     string
	}{
		{"8", "", "x86", "windows-8-x86"},
		{"8", "1", "x86", "windows-8-x86"},
		{"2008", "r2", "x64", "windows-2008r2"},
		{"2019", "", "x64", "windows-2019"},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s-%s-%s", tt.major, tt.minor, tt.arch), func(t *testing.T) {
			d, e := FromComponents("windows", tt.major, tt.minor, tt.arch)
			assert.NoError(t, e)
			assert.Equal(t, tt.expectedGcloud, d.AsGcloudArg())
		})
	}
}

func TestDistroFromComponents_ArchitectureValidation(t *testing.T) {
	var cases = []struct {
		arch, expectErrorToContain string
	}{
		{arch: "x86"},
		{arch: "x64"},
		{arch: ""},
		{arch: "mips", expectErrorToContain: "Architecture `mips` is not supported for import"},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("arch: %s", tt.arch), func(t *testing.T) {
			d, e := FromComponents("ubuntu", "18", "04", tt.arch)
			if tt.expectErrorToContain == "" {
				assert.NotNil(t, d)
				assert.NoError(t, e)
			} else {
				assert.Nil(t, d)
				assert.Error(t, e)
				assert.Contains(t, e.Error(), tt.expectErrorToContain)
			}
		})
	}
}

func TestDistroFromComponents_LinuxVersionErrors(t *testing.T) {
	var cases = []struct {
		name         string
		major, minor string
		expected     Release
		err          string
	}{
		{
			name: "major: omitted",
			err:  "major version required to be an integer greater than zero. Received: ``",
		},
		{
			name:  "major: negative",
			major: "-1",
			err:   "major version required to be an integer greater than zero. Received: `-1`",
		},
		{
			name:  "major: zero",
			major: "0",
			err:   "major version required to be an integer greater than zero. Received: `0`",
		},
		{
			name:  "major: decimal",
			major: "1.2",
			err:   "major version required to be an integer greater than zero. Received: `1.2`",
		},
		{
			name:  "major: nun-numeric",
			major: "12a",
			err:   "major version required to be an integer greater than zero. Received: `12a`",
		},
		{
			name:  "minor: negative",
			major: "10",
			minor: "-1",
			err:   "minor version required to be an integer greater than or equal to zero. Received: -1",
		},
		{
			name:  "minor: decimal",
			major: "10",
			minor: "1.5",
			err:   "minor version required to be an integer greater than or equal to zero. Received: 1.5",
		},
		{
			name:  "minor: non-numeric",
			major: "10",
			minor: "1a",
			err:   "minor version required to be an integer greater than or equal to zero. Received: 1a",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := FromComponents("centos", tt.major, tt.minor, "")
			assert.Nil(t, actual)
			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestDistroFromComponents_DistroNameErrors(t *testing.T) {
	var cases = []struct {
		distro   string
		expected Release
		err      string
	}{
		{
			err: "distro name required",
		},
		{
			distro: "a",
			err:    "distro `a` is not importable",
		},
		{
			distro: "rhel-based",
			err:    "distro `rhel-based` is not importable",
		},
		{
			distro: "unknown",
			err:    "distro `unknown` is not importable",
		},
		{
			distro: "SLESS-sap",
			err:    "distro `sless-sap` is not importable",
		},
	}
	for _, tt := range cases {
		t.Run(tt.distro, func(t *testing.T) {
			actual, err := FromComponents(tt.distro, "12", "", "")
			assert.Nil(t, actual)
			assert.EqualError(t, err, tt.err)
		})
	}
}

func TestImportCompatible(t *testing.T) {
	// Each release below is compared for compatibility.
	//   - Releases in the *same* subset are expected to be compatible with the others.
	//   - Releases in *different* subsets are expected to be incompatible
	sets := [][]Release{{
		fromID("ubuntu-1404"),
		fromComponents("ubuntu", "14", "04"),
	}, {
		fromID("ubuntu-1610"),
		fromComponents("ubuntu", "16", "10"),
	}, {
		fromID("sles-sap-12"),
		fromComponents("sles-sap", "12"),
		fromComponents("sles-sap", "12", "1"),
	}, {
		fromID("sles-sap-15"),
		fromComponents("sles-sap", "15"),
		fromComponents("sles-sap", "15", "0"),
	}, {
		fromID("sles-12"),
		fromComponents("sles", "12"),
		fromComponents("sles", "12", "5"),
	}, {
		fromID("sles-15"),
		fromComponents("sles", "15"),
		fromComponents("sles", "15", "1"),
	}, {
		fromID("centos-6"),
		fromComponents("centos", "6"),
		fromComponents("centos", "6", "1"),
	}, {
		fromID("centos-7"),
		fromComponents("centos", "7"),
		fromComponents("centos", "7", "1"),
	}, {
		fromID("rhel-7"),
		fromID("rhel-7-byol"),
		fromComponents("rhel", "7"),
		fromComponents("rhel", "7", "1"),
	}, {
		fromID("rhel-8"),
		fromID("rhel-8-byol"),
		fromComponents("rhel", "8"),
		fromComponents("rhel", "8", "1"),
	}, {
		fromID("debian-7"),
		fromComponents("debian", "7"),
		fromComponents("debian", "7", "1"),
	}, {
		fromID("debian-8"),
		fromComponents("debian", "8"),
		fromComponents("debian", "8", "1"),
	}, {
		fromID("opensuse-12"),
		fromComponents("opensuse", "12"),
		fromComponents("opensuse", "12", "4"),
	}, {
		fromID("opensuse-15"),
		fromComponents("opensuse", "15"),
		fromComponents("opensuse", "15", "4"),
	}, {
		fromID("windows-8-x86"),
		fromComponents("windows", "8", "", "x86"),
	}, {
		fromID("windows-8-x64"),
		fromComponents("windows", "8", "", "x64"),
	}, {
		fromID("windows-2008"),
		fromComponents("windows", "2008"),
	}, {
		fromID("windows-2008r2"),
		fromComponents("windows", "2008", "r2"),
	},
	}
	for i := 0; i < len(sets); i++ {
		curr := sets[i]
		for j := i; j < len(sets); j++ {
			other := sets[j]
			checkCompat(t, curr, other, i == j)
		}
	}
}

func checkCompat(t *testing.T, a []Release, b []Release, expectCompat bool) {
	for _, relA := range a {
		for _, relB := range b {
			t.Run(fmt.Sprintf("%v %v", relA.AsGcloudArg(), relB.AsGcloudArg()), func(t *testing.T) {
				assert.Equal(t, expectCompat, relA.ImportCompatible(relB))
			})
		}
	}
}

// opts allows specifying minor and architecture, in that order.
func fromComponents(distro, major string, opts ...string) Release {
	var minor, arch string
	switch len(opts) {
	case 0:
		minor, arch = "", ""
	case 1:
		minor, arch = opts[0], ""
	case 2:
		minor, arch = opts[0], opts[1]
	default:
		panic("invalid opts")
	}
	r, e := FromComponents(distro, major, minor, arch)
	if e != nil {
		panic(e)
	}
	return r
}

func fromID(id string) Release {
	r, e := FromGcloudOSArgument(id)
	if e != nil {
		panic(e)
	}
	return r
}
