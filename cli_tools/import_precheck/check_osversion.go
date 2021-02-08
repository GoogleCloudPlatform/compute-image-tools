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
	"strings"

	"github.com/GoogleCloudPlatform/osconfig/osinfo"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	daisy_utils "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/utils/daisy"
)

type osVersionCheck struct {
	osInfo *osinfo.OSInfo
}

func (c *osVersionCheck) getName() string {
	return "OS Version Check"
}

func (c *osVersionCheck) run() (*report, error) {
	r := &report{name: c.getName()}
	// Find osID from OS config's detection results.
	major, minor := splitOSVersion(c.osInfo.Version)
	osID, err := c.getOSID(major, minor, r)
	if err != nil {
		r.Info("Cannot determine OS. For supported versions, " +
			"see https://cloud.google.com/sdk/gcloud/reference/compute/images/import")
		r.skipped = true
		return r, nil
	}

	// Check whether the osID is supported for import.
	// Some systems are only available as BYOL, so check for both osID variants.
	var supported bool
	for _, suffix := range []string{"", "-byol"} {
		if daisy_utils.ValidateOS(osID+suffix) == nil {
			supported = true
			break
		}
	}
	if supported {
		if c.osInfo.ShortName == osinfo.Windows {
			// Emit the NT version for Windows, since the same NT version is
			// either Desktop or Server, and we don't want to emit a misleading message.
			r.Info(fmt.Sprintf("Detected Windows version number: NT %s", c.osInfo.Version))
		} else {
			r.Info(fmt.Sprintf("Detected system: %s", osID))
		}
	} else {
		r.Fatal(createFailureMessage(osID).Error())
	}
	return r, nil
}

func (c *osVersionCheck) getOSID(major string, minor string, r *report) (osID string, err error) {
	if c.osInfo.ShortName == osinfo.Windows {
		maj, min, err := distro.WindowsServerVersionforNTVersion(major, minor)
		if err == nil {
			major, minor = maj, min
		}
	}
	release, err := distro.FromComponents(c.osInfo.ShortName, major, minor, c.osInfo.Architecture)
	if err != nil {
		return "", err
	}
	if release == nil {
		return "", createFailureMessage("Your OS")
	}
	osID = release.AsGcloudArg()
	if osID == "" {
		return "", createFailureMessage("Your OS")
	}
	return osID, nil
}

func createFailureMessage(osID string) error {
	return fmt.Errorf("%s is not supported for import. For supported versions, "+
		"see https://cloud.google.com/sdk/gcloud/reference/compute/images/import", osID)
}

func splitOSVersion(version string) (major, minor string) {
	if version == "" {
		return "", ""
	}
	if !strings.Contains(version, ".") {
		return version, ""
	}
	parts := strings.Split(version, ".")
	return parts[0], parts[1]
}
