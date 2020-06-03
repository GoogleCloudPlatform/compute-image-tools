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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/verify"
)

const (
	centos   = "centos"
	debian   = "debian"
	opensuse = "opensuse"
	rhel     = "rhel"
	sles     = "sles"
	ubuntu   = "ubuntu"
	windows  = "windows"
)

// Release encapsulates product and version information
// about the operating systems we support.
type Release interface {
	// ImportCompatible returns whether a distro considers two releases
	// compatible. As an example, different minor versions in Ubuntu are typically
	// not compatible, while different minor versions in centOS are.
	ImportCompatible(other Release) bool

	// AsGcloudArg returns a hyphenated identifier, as used in the `--os` flag
	// of `gcloud compute images import`. It is intended for
	// showing to users in a help message. Notably, it lacks support for
	// the -byol suffix, as that is not currently modeled.
	AsGcloudArg() string
}

// FromComponents interprets the (distro, major, minor) tuple and returns
// a Release if the arguments are syntactically correct, and represent a
// release that we *may* support. The caller is responsible for verifying
// whether a translator is available elsewhere in the system.
//
// Prefer this method for generating a new Release object. It is lenient, and
// returns nice errors. The distro-specific constructors are strict and may panic.
func FromComponents(distro string, major string, minor string) (r Release, e error) {
	distro = strings.ToLower(distro)
	majorInt, e := strconv.Atoi(major)
	if e != nil || majorInt < 1 {
		return nil, fmt.Errorf(
			"major version required to be an integer greater than zero. Received: `%s`", major)
	}
	var minorInt int
	if minor == "" {
		minorInt = 0
	} else {
		minorInt, e = strconv.Atoi(minor)
		if e != nil || minorInt < 0 {
			return nil, errors.New(
				"minor version required to be an integer greater than or equal to zero. Received: " + minor)
		}
	}
	switch distro {
	case "":
		return nil, errors.New("distro name required")
	case windows:
		// We don't currently need Windows for this parsing, and punting
		// since it's not immediately clear how to represent their
		// version strings, as the user-facing value (Windows 2008r2)
		// does not match what's used internally by our tools (NT 5.2)
		// https://en.wikipedia.org/wiki/List_of_Microsoft_Windows_versions
		return r, errors.New("Windows not yet implemented")
	case ubuntu:
		return newUbuntuRelease(majorInt, minorInt)
	case centos:
		fallthrough
	case debian:
		fallthrough
	case opensuse:
		fallthrough
	case rhel:
		return newCommonLinuxRelease(distro, majorInt, minorInt)
	}

	// SLES supports variants, such as sles-sap.
	if distro == sles || strings.HasPrefix(distro, sles+"-") {
		return newSLESRelease(distro, majorInt, minorInt)
	}

	return nil, fmt.Errorf("distro `%s` is not importable", distro)
}

// FromGcloudOSArgument parses the argument provided to the `--os` flag of
// `gcloud compute images import`, and returns a Release if it represents a
// release we *may* support. The caller is responsible for verifying
// whether a translator is available elsewhere in the system.
//
// Prefer this method for generating a new Release object. It is lenient, and
// returns nice errors. The distro-specific constructors are strict and may panic.
//
// https://cloud.google.com/sdk/gcloud/reference/compute/images/import#--os
func FromGcloudOSArgument(osFlagValue string) (r Release, e error) {
	os := strings.ToLower(osFlagValue)
	if strings.HasSuffix(os, "-byol") {
		os = strings.TrimSuffix(os, "-byol")
	}
	hyphen := strings.LastIndex(os, "-")
	if hyphen < 0 || hyphen == len(os) {
		return r, fmt.Errorf("expected pattern of `distro-version`. Actual: `%s`", os)
	}
	distro, version := os[:hyphen], os[hyphen+1:]

	if distro == ubuntu {
		// In gcloud, major and minor are combined as MMmm, such as ubuntu-1804
		if len(version) != 4 {
			return r, fmt.Errorf("expected version with length four. Actual: `%s`", version)
		}

		return FromComponents("ubuntu", version[:2], version[2:])
	}
	return FromComponents(distro, version, "")
}

// commonLinuxRelease is a Release that:
//  1. Has integer major and minor versions.
//  2. Compatibility is determined by the major version.
//  3. There are no variants.
type commonLinuxRelease struct {
	distro string
	major  int
	minor  int
}

func (r commonLinuxRelease) AsGcloudArg() string {
	return fmt.Sprintf("%s-%d", r.distro, r.major)
}

func (r commonLinuxRelease) ImportCompatible(other Release) bool {
	realOther, ok := other.(commonLinuxRelease)
	return ok &&
		r.distro == realOther.distro &&
		r.major == realOther.major
}

// The caller is responsible for verifying the syntax of the arguments.
// Verify the following before calling:
//   - distro is one of {centos, debian, opensuse, rhel}
//   - major is >= 1 and minor is >= 0
func newCommonLinuxRelease(distro string, major, minor int) (Release, error) {
	verify.GreaterThanOrEqualTo(major, 1)
	verify.GreaterThanOrEqualTo(minor, 0)
	verify.Contains(distro, []string{centos, debian, opensuse, rhel})
	return commonLinuxRelease{
		distro: distro,
		major:  major,
		minor:  minor,
	}, nil
}

// slesRelease is a Release that represents the SLES distro and its variants (such as SLES for SAP).
// Compatibility requires the same variant and major version.
type slesRelease struct {
	variant string
	major   int
	minor   int
}

// The caller is responsible for verifying the syntax of the arguments.
// Verify the following before calling:
//   - distroAndVariant starts with "sles" or "sles-"
//   - major is >= 1 and minor is >= 0
//
// A non-nil error is returned if the syntax is correct, but the
// arguments do not follow SLES's naming system. Specifically:
//   - variant may not have a hyphen in its name
func newSLESRelease(distroAndVariant string, major, minor int) (Release, error) {
	verify.GreaterThanOrEqualTo(major, 1)
	verify.GreaterThanOrEqualTo(minor, 0)
	release := slesRelease{major: major, minor: minor}
	parts := strings.Split(strings.ToLower(distroAndVariant), "-")
	switch len(parts) {
	case 1:
		verify.Contains(distroAndVariant, []string{sles})
	case 2:
		verify.Contains(parts[0], []string{sles})
		release.variant = parts[1]
	default:
		return nil, fmt.Errorf("unrecognized SLES identifier: `%s`", distroAndVariant)
	}
	return release, nil
}

func (r slesRelease) ImportCompatible(other Release) bool {
	actualOther, ok := other.(slesRelease)
	return ok &&
		r.variant == actualOther.variant &&
		r.major == actualOther.major
}

func (r slesRelease) AsGcloudArg() string {
	if r.variant != "" {
		return fmt.Sprintf("sles-%s-%d", r.variant, r.major)
	}
	return fmt.Sprintf("sles-%d", r.major)
}

// ubuntuRelease is a Release that represents Ubuntu.
// Compatibility requires the same major and minor versions.
type ubuntuRelease struct {
	major int
	minor int
}

// The caller is responsible for verifying the syntax of the arguments.
// Verify the following before calling:
//   - major is >= 1 and minor is >= 0
//
// A non-nil error is returned if the syntax is correct, but the
// arguments do not follow Ubuntu's naming system. Specifically:
//   - minor version must be 4 or 10
func newUbuntuRelease(major, minor int) (Release, error) {
	verify.GreaterThanOrEqualTo(major, 1)
	verify.GreaterThanOrEqualTo(minor, 0)
	if minor == 4 || minor == 10 {
		return ubuntuRelease{major, minor}, nil
	}
	return nil, fmt.Errorf("Ubuntu version `%d.%d` is not importable", major, minor)
}

func (u ubuntuRelease) ImportCompatible(other Release) bool {
	actualOther, ok := other.(ubuntuRelease)
	return ok &&
		u.major == actualOther.major &&
		u.minor == actualOther.minor
}

func (u ubuntuRelease) AsGcloudArg() string {
	return fmt.Sprintf("ubuntu-%d%02d", u.major, u.minor)
}
