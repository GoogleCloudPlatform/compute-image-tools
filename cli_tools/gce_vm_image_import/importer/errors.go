//  Copyright 2020  Licensed under the Apache License, Version 2.0 (the "License");
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

package importer

import (
	"fmt"

	"github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/common/distro"
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

func customizeErrorToDetectionResults(osID string, w *daisy.Workflow, err error) (*daisy.Workflow, error) {
	// When translation fails, report the detection results if they don't
	// match the user's input.
	fromUser, _ := distro.ParseGcloudOsParam(osID)
	detected, _ := distro.FromLibguestfs(
		w.GetSerialConsoleOutputValue("detected_distro"),
		w.GetSerialConsoleOutputValue("detected_major_version"),
		w.GetSerialConsoleOutputValue("detected_minor_version"))
	if fromUser != nil && detected != nil && !fromUser.ImportCompatible(detected) {
		// The error is already logged by Daisy, so skipping re-logging it here.
		return w, fmt.Errorf("%q was detected on your disk, "+
			"but %q was specified. Please verify and re-import",
			detected.AsGcloudArg(), fromUser.AsGcloudArg())
	}
	return w, err
}
