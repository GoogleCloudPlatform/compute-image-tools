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
)

// customizeErrorToDetectionResults returns a custom error message when the detected OS doesn't match the detected OS.
func customizeErrorToDetectionResults(osIDFromUser string, detectionResults distro.Release, original error) error {
	fromUser, _ := distro.FromGcloudOSArgument(osIDFromUser)
	if fromUser != nil && detectionResults != nil && !fromUser.ImportCompatible(detectionResults) {
		// The error is already logged by Daisy, so skipping re-logging it here.
		return fmt.Errorf("%q was detected on your disk, "+
			"but %q was specified. Please verify and re-import",
			detectionResults.AsGcloudArg(), fromUser.AsGcloudArg())
	}
	return original
}
