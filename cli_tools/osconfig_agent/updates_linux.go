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
)

func rebootRequired() (bool, error) {
	// TODO: actually check for distro specific reboot file.
	return false, nil
}

func runUpdates() (bool, error) {
	reboot, err := rebootRequired()
	if err != nil {
		return false, err
	}
	if reboot {
		return true, nil
	}

	if err := packages.UpdatePackages(); err != nil {
		return false, err
	}

	return rebootRequired()
}
