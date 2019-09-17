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

	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/compute"
	"github.com/GoogleCloudPlatform/compute-image-tools/osconfig_tests/utils"
	computeApi "google.golang.org/api/compute/v1"
)

type patchTestSetup struct {
	testName      string
	image         string
	metadata      []*computeApi.MetadataItems
	assertTimeout time.Duration
	machineType   string
}

var (
	windowsRecordBoot = `
while ($true) {
  $uri = 'http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/boot_count'
  $old = Invoke-RestMethod -Method GET -Uri $uri -Headers @{"Metadata-Flavor" = "Google"}
  $new = $old+1
  try {
	Invoke-RestMethod -Method PUT -Uri $uri -Headers @{"Metadata-Flavor" = "Google"} -Body $new -ErrorAction Stop
  }
  catch {
	Write-Output $_.Exception.Message
	Start-Sleep 1
    continue
  }
  break
}
`
	windowsSetWsus = `
$wu_server = 'wsus-server.c.compute-image-osconfig-agent.internal'
$windows_update_path = 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate'
$windows_update_au_path = "$windows_update_path\AU"

if (Test-Connection $wu_server -Count 1 -ErrorAction SilentlyContinue) {
	if (-not (Test-Path $windows_update_path -ErrorAction SilentlyContinue)) {
		New-Item -Path $windows_update_path -Value ""
		New-Item -Path $windows_update_au_path -Value ""
	}
	Set-ItemProperty -Path $windows_update_path -Name WUServer -Value "http://${wu_server}:8530"
	Set-ItemProperty -Path $windows_update_path -Name WUStatusServer -Value "http://${wu_server}:8530"
	Set-ItemProperty -Path $windows_update_au_path -Name UseWUServer -Value 1
}
`

	linuxRecordBoot = `
uri=http://metadata.google.internal/computeMetadata/v1/instance/guest-attributes/osconfig_tests/boot_count
old=$(curl $uri -H "Metadata-Flavor: Google" -f)
new=$(($old + 1))
curl -X PUT --data "${new}" $uri -H "Metadata-Flavor: Google"
`

	enablePatch = compute.BuildInstanceMetadataItem("os-config-enabled-prerelease-features", "ospatch")

	windowsSetup = &patchTestSetup{
		assertTimeout: 60 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("sysprep-specialize-script-ps1", windowsSetWsus),
			compute.BuildInstanceMetadataItem("windows-startup-script-ps1", windowsRecordBoot+utils.InstallOSConfigGooGet()),
			enablePatch,
		},
		machineType: "n1-standard-4",
	}
	aptSetup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot+utils.InstallOSConfigDeb()),
			enablePatch,
		},
		machineType: "n1-standard-2",
	}
	el6Setup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot+utils.InstallOSConfigEL6()),
			enablePatch,
		},
		machineType: "n1-standard-2",
	}
	el7Setup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot+utils.InstallOSConfigEL7()),
			enablePatch,
		},
		machineType: "n1-standard-2",
	}
	el8Setup = &patchTestSetup{
		assertTimeout: 5 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot+utils.InstallOSConfigEL8()),
			enablePatch,
		},
		machineType: "n1-standard-2",
	}
	suseSetup = &patchTestSetup{
		assertTimeout: 10 * time.Minute,
		metadata: []*computeApi.MetadataItems{
			compute.BuildInstanceMetadataItem("startup-script", linuxRecordBoot+utils.InstallOSConfigSUSE()),
			enablePatch,
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
		el8Setup:     utils.HeadEL8Images,
		aptSetup:     utils.HeadAptImages,
		suseSetup:    utils.HeadSUSEImages,
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
		suseSetup:    utils.OldSUSEImages,
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
		el8Setup: utils.HeadEL8Images,
	}

	return imageTestSetup(mapping)
}

func suseHeadImageTestSetup() []*patchTestSetup {
	// This maps a specific patchTestSetup to test setup names and associated images.
	mapping := map[*patchTestSetup]map[string]string{
		suseSetup: utils.HeadSUSEImages,
	}

	return imageTestSetup(mapping)
}
