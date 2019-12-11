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
package main

import (
	"fmt"
	"strings"
)

var osVersions = map[string][]string{
	"centos":  {"6", "7"},
	"debian":  {"8", "9"},
	"ol":      {"6", "7"},
	"rhel":    {"6", "7"},
	"ubuntu":  {"14.04", "16.04"},
	"windows": {"6.1", "6.3", "10.0"},
}

type osVersionCheck struct{}

func (c *osVersionCheck) getName() string {
	return "OS Version Check"
}

func (c *osVersionCheck) run() (*report, error) {
	r := &report{name: c.getName()}
	var versions []string
	var ok bool
	r.Info(fmt.Sprintf("OSInfo: %+v", osInfo))

	if versions, ok = osVersions[osInfo.ShortName]; !ok {
		r.Fatal(osInfo.ShortName)
		return r, nil
	}

	found := false
	for _, version := range versions {
		if strings.Contains(osInfo.Version, version) {
			found = true
			break
		}
	}
	if !found {
		r.Fatal(fmt.Sprintf("version: %q not supported", osInfo.Version))
	}
	return r, nil
}
