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

package main

import (
	"os/exec"

	osconfigpb "github.com/GoogleCloudPlatform/compute-image-tools/cli_tools/osconfig_agent/_internal/gapi-cloud-osconfig-go/google.golang.org/genproto/googleapis/cloud/osconfig/v1alpha1"
	"github.com/GoogleCloudPlatform/compute-image-tools/package_library"
	"golang.org/x/sys/windows/registry"
)

func rebootRequired() (bool, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	k.Close()

	return true, nil
}

classifications = map[osconfig.WindowsUpdateSettings_Classification]string{
	osconfigpb.WindowsUpdateSettings_CRITICAL: "",
	osconfigpb.WindowsUpdateSettings_SECURITY: "",
	osconfigpb.WindowsUpdateSettings_DEFINITION: "",
	osconfigpb.WindowsUpdateSettings_DRIVER: "",
	osconfigpb.WindowsUpdateSettings_FEATURE_PACK: "",
	osconfigpb.WindowsUpdateSettings_SERVICE_PACK: "",
	osconfigpb.WindowsUpdateSettings_TOOL: "",
	osconfigpb.WindowsUpdateSettings_UPDATE_ROLLUP: "",
	osconfigpb.WindowsUpdateSettings_UPDATE: "",
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

	query := "IsInstalled=0"
	for _, c := range pp.WindowsUpdate.Classifications {
		sc, ok := classifications[c]
		if !ok {
			fmt.Println("Unknown classification:", c)
			continue
		}
		query = fmt.Sprintf("%s and Type=%s", query, sc) 
	}
	for _, e := range pp.WindowsUpdate.Excludes {
		query = fmt.Sprintf("%s and UpdateID!=%s", query, e) 
	}

	if err := packages.InstallWUAUpdates(query); err != nil {
		return false, err
	}

	if packages.GooGetExists {
		if err := packages.InstallGooGetUpdates(); err != nil {
			return false, err
		}
	}

	if pp.RebootConfig != osconfigpb.PatchPolicy_NEVER {
		return rebootRequired()
	}
	return false, nil
}

func rebootSystem() error {
	return exec.Command("shutdown", "/r", "/t", "00", "/f", "/d", "p:2:3").Run()
}
