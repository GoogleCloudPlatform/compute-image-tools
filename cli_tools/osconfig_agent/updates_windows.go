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
	"github.com/GoogleCloudPlatform/compute-image-tools/package_library"
	"github.com/google/logger"
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

func runUpdates() (bool, error) {
	reboot, err := rebootRequired()
	if err != nil {
		false, err
	}
	if reboot {
		true, nil
	}

	if err := packages.InstallWUAUpdates("IsInstalled=0"); err != nil {
		logger.Errorln("Error installing Windows updates:", err)
	}

	if packages.GooGetExists {
		if err := packages.InstallGooGetUpdates(); err != nil {
			logger.Errorln("Error installing GooGet updates:", err)
		}
	}

	return rebootRequired()
}
