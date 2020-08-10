//  Copyright 2020 Google Inc. All Rights Reserved.
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

package ovfexporter

import "github.com/vmware/govmomi/ovf"

// PopulateOS populates OS info in OVF descriptor.
func PopulateOS(descriptor *ovf.Envelope) error {
	var id int16
	var osType, version string

	//TODO
	id = 94
	osType = "ubuntu64Guest"
	version = "14.04"

	descriptor.OperatingSystem = &ovf.OperatingSystemSection{ID: id, OSType: &osType, Version: &version}
	descriptor.OperatingSystem.Info = "The kind of installed guest operating system"
	return nil
}
