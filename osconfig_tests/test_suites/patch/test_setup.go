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

var (
	windowsRecordBoot = `
$uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/install_done'
$old = Invoke-RestMethod -Method GET -Uri $uri -Headers @{"Metadata-Flavor" = "Google"}
$new = $old+1
Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body $new`
	windowsStartup = utils.InstallOSConfigGooGet + windowsRecordBoot

	linuxRecordBoot = `
uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/boot_count
old=$(curl $uri -H "Metadata-Flavor: Google" -f)
new=$(($old + 1))
curl -X PUT --data "${new}" $uri -H "Metadata-Flavor: Google"`
	aptStartup = utils.InstallOSConfigDeb + linuxRecordBoot
	el6Startup = utils.InstallOSConfigYumEL6 + linuxRecordBoot
	el7Startup = utils.InstallOSConfigYumEL7 + linuxRecordBoot

	windowsSetup = &patchTestSetup{
		assertTimeout: 30 * time.Minute,
		startup: &compute.MetadataItems{
			Key:   "windows-startup-script-ps1",
			Value: &windowsStartup,
		},
		machineType: "n1-standard-4",
	}
	aptSetup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		startup: &compute.MetadataItems{
			Key:   "startup-script",
			Value: &aptStartup,
		},
		machineType: "n1-standard-2",
	}
	el6Setup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		startup: &compute.MetadataItems{
			Key:   "startup-script",
			Value: &el6Startup,
		},
		machineType: "n1-standard-2",
	}
	el7Setup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		startup: &compute.MetadataItems{
			Key:   "startup-script",
			Value: &el7Startup,
		},
		machineType: "n1-standard-2",
	}
)

func imageTestSetup(mapping map[*patchTestSetup]map[string]string) (setup []*patchTestSetup) {
	for s, m := range mapping {
		for name, image := range m {
			new := patchTestSetup(*s)
			new.testName = name
			new.image = image
			setup = append(setup, &new)
		}
	}
	return
}

func headImageTestSetup() []*patchTestSetup {
	// This maps a specific patchTestSetup to test setup names and associated images.
	mapping := map[*patchTestSetup]map[string]string{
		windowsSetup: utils.HeadWindowsImages,
		el6Setup:     utils.HeadEL6Images,
		el7Setup:     utils.HeadEL7Images,
		aptSetup:     utils.HeadAptImages,
	}

	return imageTestSetup(mapping)
}

func oldImageTestSetup() []*patchTestSetup {
	// This maps a specific patchTestSetup to test setup names and associated images.
	mapping := map[*patchTestSetup]map[string]string{
		windowsSetup: utils.OldWindowsImages,
		el6Setup:     utils.OldEL6Images,
		el7Setup:     utils.OldEL7Images,
		aptSetup:     utils.OldAptImages,
	}

	return imageTestSetup(mapping)
}

func aptHeadImageTestSetup() []*patchTestSetup {
	// This maps a specific patchTestSetup to test setup names and associated images.
	mapping := map[*patchTestSetup]map[string]string{
		aptSetup: utils.HeadAptImages,
	}

	return imageTestSetup(mapping)
}

func yumHeadImageTestSetup() []*patchTestSetup {
	// This maps a specific patchTestSetup to test setup names and associated images.
	mapping := map[*patchTestSetup]map[string]string{
		el6Setup: utils.HeadEL6Images,
		el7Setup: utils.HeadEL7Images,
	}

	return imageTestSetup(mapping)
}
