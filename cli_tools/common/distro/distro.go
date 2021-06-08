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
	"regexp"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/assert"
)

const (
	centos   = "centos"
	debian   = "debian"
	opensuse = "opensuse"
	rhel     = "rhel"
	sles     = "sles"
	slesSAP  = "sles-sap"
	ubuntu   = "ubuntu"
	windows  = "windows"

	archX86 = "x86"
	archX64 = "x64"

	r2 = "r2"
)

var osFlagExpression = regexp.MustCompile(
	// Distro is required and is comprised of at least one letter.
	// There may be two segments, separated by a hyphen.
	// Examples: ubuntu, sles, sles-sap, windows
	"^(?P<distro>[a-z]+(?:-[a-z]+)?)" +
		// Version is required, and is at least one word character.
		// Examples: 2004, 2008, 2008r2
		"-(?P<version>[a-z0-9]+)" +
		// Architecture is optional. The only options are `x86` and `x64`.
		"(?:-(?P<arch>x86|x64))?" +
		// License is optional. The only value is `byol`.
		"(?:-(?P<license>byol))?$")

// standardArchitectures contains the architectures that we support, along with possible synonyms.
var standardArchitectures = []struct {
	arch     string
	synonyms []string
}{
	{
		archX86,
		[]string{"amd64", "x86_64"},
	}, {
		archX64,
		[]string{"i386", "i686", "x86_32"},
	},
}

// Flags that don't follow `osFlagExpression` and have been
// replaced with newer versions. Mapping is from legacy to modern.
var legacyFlags = map[string]string{
	// windows-8-1-x64-byol includes an extra hyphen between its
	// major and minor version. The non-legacy flag is windows-8-x64-byol.
	"windows-8-1-x64-byol": "windows-8-x64-byol",
}

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
func FromComponents(distro string, major string, minor string, architecture string) (r Release, e error) {
	standardArch, err := standardizeArchitecture(architecture)
	if err != nil {
		return nil, err
	}
	standardDistro, err := standardizeDistro(distro)
	if err != nil {
		return nil, err
	}

	if standardDistro == windows {
		return newWindowsRelease(major, minor, standardArch)
	}

	return newLinuxRelease(standardDistro, major, minor)
}

// standardizeArchitecture maps a raw string to a known architecture. It's not an error to
// have an empty architecture, but if it's specified, it has to match one of the
// architectures in standardArchitectures.
func standardizeArchitecture(architecture string) (string, error) {
	if architecture == "" {
		return "", nil
	}
	lowered := strings.ToLower(architecture)
	for _, standardArch := range standardArchitectures {
		if standardArch.arch == lowered {
			return standardArch.arch, nil
		}
		for _, synonym := range standardArch.synonyms {
			if synonym == lowered {
				return standardArch.arch, nil
			}
		}
	}
	return "", fmt.Errorf("Unrecognized architecture `%s`", architecture)
}

// standardizeDistro maps a raw string to a known distro.
// It's an error if distro is empty, or if a match isn't found.
func standardizeDistro(distro string) (string, error) {
	if distro == "" {
		return "", errors.New("distro name required")
	}
	d := strings.ReplaceAll(strings.ToLower(distro), "_", "-")
	for _, known := range []string{centos, debian, opensuse, rhel, slesSAP, sles, ubuntu, windows} {
		if strings.Contains(d, known) {
			return known, nil
		}
	}
	return "", fmt.Errorf("Unrecognized distro `%s`", distro)
}

func newLinuxRelease(distro string, major string, minor string) (Release, error) {
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
	case sles:
		fallthrough
	case slesSAP:
		return newSLESRelease(distro, majorInt, minorInt)
	default:
		return nil, fmt.Errorf("Unrecognized distro `%s`", distro)
	}
}

// FromGcloudOSArgumentMustParse parses the argument provided to the `--os` flag of
// `gcloud compute images import`, and returns a Release if it represents a
// release we *may* support. If osFlagValue does not parse, the call panics.
func FromGcloudOSArgumentMustParse(osFlagValue string) Release {
	r, err := FromGcloudOSArgument(osFlagValue)
	if err != nil {
		panic(err)
	}
	return r
}

// FromGcloudOSArgument parses the argument provided to the `--os` flag of
// `gcloud compute images import`, and returns a Release if it represents a
// release we *may* support. The caller is responsible for verifying
// whether a translator is available elsewhere in the system.
//
// https://cloud.google.com/sdk/gcloud/reference/compute/images/import#--os
func FromGcloudOSArgument(osFlagValue string) (r Release, e error) {
	if legacyFlags[osFlagValue] != "" {
		osFlagValue = legacyFlags[osFlagValue]
	}
	match := osFlagExpression.FindStringSubmatch(strings.ToLower(osFlagValue))
	if match == nil {
		return r, fmt.Errorf("expected pattern of `distro-version`. Actual: `%s`", osFlagValue)
	}
	components := make(map[string]string)
	for i, name := range osFlagExpression.SubexpNames() {
		if i != 0 && name != "" {
			components[name] = match[i]
		}
	}
	distro, version, arch := components["distro"], components["version"], components["arch"]

	var major, minor string
	if distro == ubuntu {
		// In gcloud, major and minor are combined as MMmm, such as ubuntu-1804
		if len(version) != 4 {
			return r, fmt.Errorf("expected version with length four. Actual: `%s`", version)
		}
		major, minor = version[:2], version[2:]
	} else if distro == windows && strings.HasSuffix(version, r2) {
		major, minor = version[:len(version)-2], r2
	} else {
		major, minor = version, ""
	}
	return FromComponents(distro, major, minor, arch)
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

func commonLinuxDistros() []string {
	return []string{centos, debian, opensuse, rhel}
}

// The caller is responsible for verifying the syntax of the arguments.
// Verify the following before calling:
//   - distro is one of the distros returned by commonLinuxDistros().
//   - major is >= 1 and minor is >= 0
func newCommonLinuxRelease(distro string, major, minor int) (Release, error) {
	assert.GreaterThanOrEqualTo(major, 1)
	assert.GreaterThanOrEqualTo(minor, 0)
	assert.Contains(distro, commonLinuxDistros())
	return commonLinuxRelease{
		distro: distro,
		major:  major,
		minor:  minor,
	}, nil
}

// windowsRelease uses marketing versions rather than NT versions.
// Currently the only minor version is "r2". For example, the versions
// 2012 and 2012r2 and *not* import compatible, and have different
// minor versions. In the first case, the minor version is empty.
// In the second it is "r2".
//
// In terms of import compatibility, currently r2 is the only minor version
// that is treated separately. Other minor versions (eg: 8 vs 8.1) are treated
// imported using the same logic.
type windowsRelease struct {
	major, minor, architecture string
}

func (w windowsRelease) ImportCompatible(other Release) bool {
	actualOther, ok := other.(windowsRelease)
	compatible := ok &&
		w.major == actualOther.major &&
		w.architecture == actualOther.architecture
	if w.minor == r2 || actualOther.minor == r2 {
		compatible = compatible && (w.minor == actualOther.minor)
	}
	return compatible
}

func (w windowsRelease) AsGcloudArg() string {
	arg := fmt.Sprintf("windows-%s", w.major)
	if w.minor == r2 {
		arg += r2
	}
	// Architecture is only included for desktop versions (eg: 7, 8, 10);
	// Server versions are assumed to be x64, so that's not included in the
	// osID.
	if w.architecture != "" && len(w.major) <= 2 {
		arg += "-" + w.architecture
	}
	return arg
}

func newWindowsRelease(major string, minor string, architecture string) (Release, error) {
	if !regexp.MustCompile("^\\d+$").MatchString(major) {
		return nil, fmt.Errorf("`%s` is not a valid major version for Windows", major)
	}
	return windowsRelease{major, minor, architecture}, nil
}

// WindowsServerVersionforNTVersion returns the marketing version corresponding to an NT version.
//
// Careful: NT versions are shared across multiple Windows products, so this function is a
// rough heuristic. It follows these rules for a collision:
//   - Only returns server versions (not desktop).
//   - If multiple server versions are possible, return the earliest.
func WindowsServerVersionforNTVersion(major string, minor string) (marketingMajor, marketingMinor string, err error) {
	// Mappings of NT version to marketing versions.
	// Source: https://wikipedia.org/wiki/List_of_Microsoft_Windows_versions
	for _, t := range []struct {
		ntMajor        string
		ntMinor        string
		marketingMajor string
		marketingMinor string
	}{
		{"6", "0", "2008", ""},
		{"6", "1", "2008", "r2"},
		{"6", "2", "2012", ""},
		{"6", "3", "2012", "r2"},
		{"10", "0", "2016", ""}, // NT 10.0 is also 2019
	} {
		if major == t.ntMajor && minor == t.ntMinor {
			return t.marketingMajor, t.marketingMinor, nil
		}
	}
	return "", "", fmt.Errorf("`%s.%s` is not a recognized Windows NT version", major, minor)
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
	assert.GreaterThanOrEqualTo(major, 1)
	assert.GreaterThanOrEqualTo(minor, 0)
	var variant string
	switch distroAndVariant {
	case slesSAP:
		variant = "sap"
	case sles:
		variant = ""
	default:
		panic(fmt.Sprintf("%q is not valid for SLES", distroAndVariant))
	}
	return slesRelease{variant, major, minor}, nil
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
	assert.GreaterThanOrEqualTo(major, 1)
	assert.GreaterThanOrEqualTo(minor, 0)
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
