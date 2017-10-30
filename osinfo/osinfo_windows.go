/*
Copyright 2017 Google Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package osinfo

import (
	"github.com/StackExchange/wmi"
)

func GetDistributionInfo() (*DistributionInfo, error) {
	oi, err := osInfo()
	if err != nil {
		return nil, err
	}

	// TODO(ajackura): Get kernel version from ntoskrnl.exe.
	return &DistributionInfo{ShortName: "windows", LongName: oi.Caption, Version: oi.Version, kernel: oi.Version}, nil
}

type win32_OperatingSystem struct {
	Caption, Version string
}

func osInfo() (*win32_OperatingSystem, error) {
	var ops []win32_OperatingSystem
	if err := wmi.Query(wmi.CreateQuery(&ops, ""), &ops); err != nil {
		return nil, err
	}
	return &ops[0], nil
}
