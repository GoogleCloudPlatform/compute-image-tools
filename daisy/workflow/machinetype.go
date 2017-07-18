//  Copyright 2017 Google Inc. All Rights Reserved.
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

package workflow

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var machineTypeURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[1]s)/machineTypes/(?P<machinetype>%[1]s)$`, rfc1035))

var machineTypes struct {
	valid []string
	mu    sync.Mutex
}

func checkMachineType(client compute.Client, project, zone, machineType string) error {
	machineTypes.mu.Lock()
	defer machineTypes.mu.Unlock()
	url := fmt.Sprintf("/project/%s/zone/%s/machinetype/%s", project, zone, machineType)
	if strIn(url, machineTypes.valid) {
		return nil
	}
	if _, err := client.GetMachineType(project, zone, machineType); err != nil {
		return err
	}
	machineTypes.valid = append(machineTypes.valid, url)
	return nil
}
