//  Copyright 2019 Google Inc. All Rights Reserved.
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
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	"google.golang.org/api/compute/v1"
)

type patchTestSetup struct {
	testName      string
	image         string
	startup       *compute.MetadataItems
	assertTimeout time.Duration
	machineType   string
}

var windowsSetup = &patchTestSetup{
	assertTimeout: 15 * time.Minute,
	startup: &compute.MetadataItems{
		Key:   "windows-startup-script-ps1",
		Value: &utils.InstallOSConfigGooGet,
	},
	machineType: "n1-standard-4",
}

var aptSetup = &patchTestSetup{
	assertTimeout: 5 * time.Minute,
	startup: &compute.MetadataItems{
		Key:   "startup-script",
		Value: &utils.InstallOSConfigDeb,
	},
	machineType: "n1-standard-2",
}

var el6Setup = &patchTestSetup{
	assertTimeout: 5 * time.Minute,
	startup: &compute.MetadataItems{
		Key:   "startup-script",
		Value: &utils.InstallOSConfigYumEL6,
	},
	machineType: "n1-standard-2",
}

var el7Setup = &patchTestSetup{
	assertTimeout: 5 * time.Minute,
	startup: &compute.MetadataItems{
		Key:   "startup-script",
		Value: &utils.InstallOSConfigYumEL7,
	},
	machineType: "n1-standard-2",
}

func headImageTestSetup() (setup []*patchTestSetup) {
	// This maps a specific patchTestSetup to test setup names and associated images.
	headTestSetupMapping := map[*patchTestSetup]map[string]string{
		windowsSetup: utils.HeadWindowsImages,
		el6Setup:     utils.HeadEL6Images,
		el7Setup:     utils.HeadEL7Images,
		aptSetup:     utils.HeadAptImages,
	}

	for s, m := range headTestSetupMapping {
		for name, image := range m {
			new := patchTestSetup(*s)
			new.testName = name
			new.image = image
			setup = append(setup, &new)
		}
	}
	return
}
