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
	"strings"
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
// about a Linux or Windows release.
type Release interface {
	ImportCompatible(other Release) bool
	AsGcloudArg() string
}

type release struct {
	distro  string
	variant string
	major   string
	minor   string
}

// AsGcloudArg returns a string representation using the format
// that we use with the `--os` flag.
func (l release) AsGcloudArg() string {
	switch l.distro {
	case ubuntu:
		return fmt.Sprintf("%s-%s%s", l.distro, l.major, l.minor)
	case sles:
		if l.variant != "" {
			return fmt.Sprintf("%s-%s-%s", l.distro, l.variant, l.major)
		}
		return fmt.Sprintf("%s-%s", l.distro, l.major)
	default:
		return fmt.Sprintf("%s-%s", l.distro, l.major)
	}
}

// ImportCompatible reports whether two releases are compatible for import.
func (l release) ImportCompatible(other Release) bool {
	realOther, ok := other.(release)
	if !ok {
		return false
	}

	if l.distro == ubuntu {
		return l.distro == realOther.distro &&
			l.major == realOther.major &&
			l.minor == realOther.minor
	}

	if l.distro == sles {
		return l.distro == realOther.distro &&
			l.variant == realOther.variant &&
			l.major == realOther.major
	}

	return l.distro == realOther.distro &&
		l.major == realOther.major
}

// FromLibguestfs initializes a DistroAndVersion from the fields returned by libguestfs's
// inspection routines.
//
// http://libguestfs.org/guestfs.3.html#guestfs_inspect_get_distro
func FromLibguestfs(distro string, major string, minor string) (r Release, e error) {
	distro = strings.ToLower(distro)
	if distro == windows {
		// We don't currently need Windows for this parsing, and punting
		// since it's not immediately clear how to represent their
		// version strings, as the user-facing value (Windows 2008r2)
		// does not match what's used internally by our tools (NT 5.2)
		// https://en.wikipedia.org/wiki/List_of_Microsoft_Windows_versions
		return r, errors.New("Windows not yet implemented")
	}
	if distro == "" || major == "" {
		return nil, errors.New("distro and major are required")
	}
	return release{
		distro: distro,
		major:  major,
		minor:  minor,
	}, nil
}

// ParseGcloudOsParam parses the value associated with the `--os` parameter
// from the `gcloud compute image` tools.
//
// https://cloud.google.com/sdk/gcloud/reference/compute/images/import#--os
func ParseGcloudOsParam(osFlagValue string) (r Release, e error) {
	os := strings.ToLower(osFlagValue)
	if strings.HasSuffix(os, "-byol") {
		os = strings.TrimSuffix(os, "-byol")
	}
	if strings.HasPrefix(os, windows) {
		// We don't currently need Windows for this parsing, and punting
		// since it's not immediately clear how to represent their
		// version strings, as the user-facing value (Windows 2008r2)
		// does not match what's used internally by our tools (NT 5.2)
		// https://en.wikipedia.org/wiki/List_of_Microsoft_Windows_versions
		return r, errors.New("Windows not yet implemented")
	}
	parts := strings.Split(os, "-")
	distro := parts[0]

	// 1. Format is `(debian|centos|rhel|opensuse)-major`.
	//   Canonical example: debian-8
	if distro == debian ||
		distro == centos ||
		distro == rhel ||
		distro == opensuse {
		if len(parts) != 2 {
			return r, fmt.Errorf("expected pattern of `distro-version`. Actual: `%s`", os)
		}

		return release{
			distro: distro,
			major:  parts[1],
		}, nil
	}

	// 2. Format is `sles(-variant)?-major`.
	//   Canonical examples: sles-sap-12, sles-12
	if distro == sles {
		switch len(parts) {
		case 2:
			return release{
				distro: distro,
				major:  parts[1],
			}, nil
		case 3:
			return release{
				distro:  distro,
				variant: parts[1],
				major:   parts[2],
			}, nil
		default:
			return r, fmt.Errorf("unrecognized SLES identifier: `%s`", os)
		}
	}

	// 3. Format is `ubuntu-(major)(minor)`.
	//   Canonical example: ubuntu-1804
	if distro == ubuntu {
		if len(parts) != 2 {
			return r, fmt.Errorf("expected pattern of `distro-version`. Actual: `%s`", os)
		}

		version := parts[1]
		// We support importing LTS versions, which end in 04.
		if len(version) != 4 {
			return r, fmt.Errorf("expected version with length four. Actual: `%s`", version)
		}

		return release{
			distro: distro,
			major:  version[:2],
			minor:  version[2:],
		}, nil
	}

	return r, fmt.Errorf("unrecognized identifier: `%s`", os)
}
