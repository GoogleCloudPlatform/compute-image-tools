//  Copyright 2018 Google Inc. All Rights Reserved.
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

package patch

import (
	"os/exec"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/google-osconfig-agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/go/packages"
)

func rebootRequired() (bool, error) {
	// TODO: actually check for distro specific reboot file.
	return false, nil
}

func runUpdates(pp *patchPolicy) (bool, error) {
	if pp.RebootConfig != osconfigpb.PatchPolicy_NEVER {
		reboot, err := rebootRequired()
		if err != nil {
			return false, err
		}
		if reboot {
			return true, nil
		}
	}

	if err := packages.UpdatePackages(); err != nil {
		return false, err
	}

	if pp.RebootConfig != osconfigpb.PatchPolicy_NEVER {
		return rebootRequired()
	}
	return false, nil
}

func rebootSystem() error {
	// TODO: make this work on all systems, maybe fall back to reboot(2)
	return exec.Command("shutdown", "-r", "-t", "0").Run()
}
