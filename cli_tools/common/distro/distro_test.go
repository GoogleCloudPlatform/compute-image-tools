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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGcloudOsParam_WindowsIsNotImplemented(t *testing.T) {
	d, err := ParseGcloudOsParam("windows-2008")
	assert.Nil(t, d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "windows not yet implemented")
}

func TestFromLibguestfs_WindowsIsNotImplemented(t *testing.T) {
	d, err := FromLibguestfs("windows", "6", "1")
	assert.Nil(t, d)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "windows not yet implemented")
}

func TestParseGcloudOsParam_HappyCasesLinux(t *testing.T) {
	var cases = []struct {
		in       string
		expected release
	}{
		{"debian-8", release{distro: "debian", major: "8"}},
		{"debian-9", release{distro: "debian", major: "9"}},
		{"centos-6", release{distro: "centos", major: "6"}},
		{"centos-7", release{distro: "centos", major: "7"}},
		{"centos-8", release{distro: "centos", major: "8"}},
		{"opensuse-15", release{distro: "opensuse", major: "15"}},
		{"sles-sap-12-byol", release{distro: "sles", variant: "sap", major: "12"}},
		{"sles-12-byol", release{distro: "sles", major: "12"}},
		{"sles-15-byol", release{distro: "sles", major: "15"}},
		{"rhel-6", release{distro: "rhel", major: "6"}},
		{"rhel-7", release{distro: "rhel", major: "7"}},
		{"rhel-8", release{distro: "rhel", major: "8"}},
		{"rhel-6-byol", release{distro: "rhel", major: "6"}},
		{"rhel-7-byol", release{distro: "rhel", major: "7"}},
		{"rhel-8-byol", release{distro: "rhel", major: "8"}},
		{"ubuntu-1404", release{distro: "ubuntu", major: "14", minor: "04"}},
		{"ubuntu-1604", release{distro: "ubuntu", major: "16", minor: "04"}},
		{"ubuntu-1804", release{distro: "ubuntu", major: "18", minor: "04"}},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			d, e := ParseGcloudOsParam(tt.in)
			assert.NoError(t, e)
			assert.Equal(t, tt.expected, d)
		})
	}
}

func TestHumanReadable(t *testing.T) {
	var cases = []struct {
		in       release
		expected string
	}{
		{release{distro: "debian", major: "8"}, "debian-8"},
		{release{distro: "opensuse", major: "15"}, "opensuse-15"},
		{release{distro: "sles", variant: "sap", major: "12"}, "sles-sap-12"},
		{release{distro: "sles", major: "15", minor: "1"}, "sles-15"},
		{release{distro: "ubuntu", major: "14", minor: "04"}, "ubuntu-1404"},
	}
	for _, tt := range cases {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.in.AsGcloudArg())
		})
	}
}

func TestParseGcloudOsParam_ErrorsLinux(t *testing.T) {
	var cases = []struct {
		in  string
		err string
	}{
		{"", "unrecognized identifier: ``"},
		{"notSupported", "unrecognized identifier: `notsupported`"},
		{"notSupported-18", "unrecognized identifier: `notsupported-18`"},
		{"notSupported-1804", "unrecognized identifier: `notsupported-1804`"},
		{"sles", "unrecognized SLES identifier: `sles`"},
		{"kali-12", "unrecognized identifier: `kali-12`"},
		{"ubuntu", "expected pattern of `distro-version`"},
		{"ubuntu-12", "expected version with length four"},
		{"opensuse-15-leap", "expected pattern of `distro-version`"},
		{"debian", "expected pattern of `distro-version`"},
		{"centos7", "unrecognized identifier: `centos7`"},
		{"rhel", "expected pattern of `distro-version`"},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			d, e := ParseGcloudOsParam(tt.in)
			assert.Nil(t, d)
			assert.Error(t, e)
			assert.Contains(t, e.Error(), tt.err)
		})
	}
}

func TestImportCompatible_Libguestfs(t *testing.T) {
	assert.True(t,
		safeParse(t, "centos-7").ImportCompatible(
			safeFromLibguestfs(t, "centos", "7", "")))

	assert.True(t,
		safeParse(t, "centos-7").ImportCompatible(
			safeFromLibguestfs(t, "centos", "7", "1")))

	assert.True(t,
		safeParse(t, "ubuntu-1404").ImportCompatible(
			safeFromLibguestfs(t, "ubuntu", "14", "04")))

	assert.False(t,
		safeParse(t, "rhel-7").ImportCompatible(
			safeFromLibguestfs(t, "centos", "7", "")))

	assert.False(t,
		safeParse(t, "centos-7").ImportCompatible(
			safeFromLibguestfs(t, "debian", "7", "")))

	assert.False(t,
		safeParse(t, "ubuntu-1404").ImportCompatible(
			safeFromLibguestfs(t, "ubuntu", "14", "10")))
}

func TestImportCompatible_CommonLinux(t *testing.T) {
	cent7 := safeParse(t, "centos-7")
	cent8 := safeParse(t, "centos-8")
	rhel7 := safeParse(t, "rhel-7")
	deb7 := safeParse(t, "debian-7")

	assert.True(t, cent7.ImportCompatible(cent7))
	assert.False(t, cent7.ImportCompatible(cent8))
	assert.False(t, cent7.ImportCompatible(rhel7))
	assert.False(t, cent7.ImportCompatible(deb7))
}

func TestImportCompatible_Ubuntu(t *testing.T) {
	ubuntu1404 := release{distro: "ubuntu", major: "14", minor: "04"}
	ubuntu1410 := release{distro: "ubuntu", major: "14", minor: "10"}

	assert.True(t, ubuntu1404.ImportCompatible(ubuntu1404))
	assert.True(t, ubuntu1410.ImportCompatible(ubuntu1410))

	assert.False(t, ubuntu1404.ImportCompatible(ubuntu1410))
}

func TestImportCompatible_SLES(t *testing.T) {
	opensuse := release{distro: "opensuse", major: "15"}
	sles := release{distro: "sles", major: "15"}
	slesSap := release{distro: "sles", variant: "sap", major: "15"}

	assert.True(t, opensuse.ImportCompatible(opensuse))
	assert.True(t, sles.ImportCompatible(sles))
	assert.True(t, slesSap.ImportCompatible(slesSap))

	assert.False(t, sles.ImportCompatible(opensuse))
	assert.False(t, sles.ImportCompatible(slesSap))
	assert.False(t, slesSap.ImportCompatible(opensuse))
}

func safeParse(t *testing.T, s string) Release {
	r, e := ParseGcloudOsParam(s)
	assert.NoError(t, e)
	return r
}

func safeFromLibguestfs(t *testing.T, distro, major, minor string) Release {
	r, e := FromLibguestfs(distro, major, minor)
	assert.NoError(t, e)
	return r
}
