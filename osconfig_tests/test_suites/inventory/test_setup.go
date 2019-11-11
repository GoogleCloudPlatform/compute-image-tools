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

package inventory

import (
	"strings"

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	computeAPI "google.golang.org/api/compute/v1"
)

type inventoryTestSetup struct {
	testName    string
	hostname    string
	image       string
	packageType []string
	shortName   string
	startup     *computeAPI.MetadataItems
	machineType string
}

var (
	windowsSetup = &inventoryTestSetup{
		packageType: []string{"googet", "wua", "qfe"},
		shortName:   "windows",

		startup:     compute.BuildInstanceMetadataItem("windows-startup-script-ps1", utils.InstallOSConfigGooGet()),
		machineType: "n1-standard-4",
	}

	aptSetup = &inventoryTestSetup{
		packageType: []string{"deb"},
		startup:     compute.BuildInstanceMetadataItem("startup-script", utils.InstallOSConfigDeb()),
		machineType: "n1-standard-2",
	}

	el6Setup = &inventoryTestSetup{
		packageType: []string{"rpm"},
		startup:     compute.BuildInstanceMetadataItem("startup-script", utils.InstallOSConfigEL6()),
		machineType: "n1-standard-2",
	}

	el7Setup = &inventoryTestSetup{
		packageType: []string{"rpm"},
		startup:     compute.BuildInstanceMetadataItem("startup-script", utils.InstallOSConfigEL7()),
		machineType: "n1-standard-2",
	}

	el8Setup = &inventoryTestSetup{
		packageType: []string{"rpm"},
		startup:     compute.BuildInstanceMetadataItem("startup-script", utils.InstallOSConfigEL8()),
		machineType: "n1-standard-2",
	}

	suseSetup = &inventoryTestSetup{
		packageType: []string{"zypper"},
		startup:     compute.BuildInstanceMetadataItem("startup-script", utils.InstallOSConfigSUSE()),
		machineType: "n1-standard-2",
	}
)

func headImageTestSetup() (setup []*inventoryTestSetup) {
	// This maps a specific inventoryTestSetup to test setup names and associated images.
	headTestSetupMapping := map[*inventoryTestSetup]map[string]string{
		windowsSetup: utils.HeadWindowsImages,
		el6Setup:     utils.HeadEL6Images,
		el7Setup:     utils.HeadEL7Images,
		el8Setup:     utils.HeadEL8Images,
		aptSetup:     utils.HeadAptImages,
		suseSetup:    utils.HeadSUSEImages,
	}

	for s, m := range headTestSetupMapping {
		for name, image := range m {
			new := inventoryTestSetup(*s)
			new.testName = name
			new.image = image
			if strings.Contains(name, "centos") {
				new.shortName = "centos"
			} else if strings.Contains(name, "rhel") {
				new.shortName = "rhel"
			} else if strings.Contains(name, "debian") {
				new.shortName = "debian"
			} else if strings.Contains(name, "ubuntu") {
				new.shortName = "ubuntu"
			} else if strings.Contains(name, "sles") {
				new.shortName = "sles"
			} else if strings.Contains(name, "opensuse-leap") {
				new.shortName = "opensuse-leap"
			}
			setup = append(setup, &new)
		}
	}
	return
}
