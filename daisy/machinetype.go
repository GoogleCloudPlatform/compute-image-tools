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

package daisy

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var machineTypeURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/machineTypes/(?P<machinetype>%[2]s)$`, projectRgxStr, rfc1035))

var machineTypeCache struct {
	exists map[string]map[string][]string
	mu     sync.Mutex
}

func machineTypeExists(client compute.Client, project, zone, machineType string) (bool, dErr) {
	machineTypeCache.mu.Lock()
	defer machineTypeCache.mu.Unlock()
	if machineTypeCache.exists == nil {
		machineTypeCache.exists = map[string]map[string][]string{}
	}
	if _, ok := machineTypeCache.exists[project]; !ok {
		machineTypeCache.exists[project] = map[string][]string{}
	}
	if _, ok := machineTypeCache.exists[project][zone]; !ok {
		mtl, err := client.ListMachineTypes(project, zone)
		if err != nil {
			return false, errf("error listing machine types for project %q: %v", project, err)
		}
		var mts []string
		for _, mt := range mtl {
			mts = append(mts, mt.Name)
		}
		machineTypeCache.exists[project][zone] = mts
	}
	if strIn(machineType, machineTypeCache.exists[project][zone]) {
		return true, nil
	}
	// Check for custom machine types.
	if _, err := client.GetMachineType(project, zone, machineType); err != nil {
		return false, typedErr(apiError, err)
	}
	machineTypeCache.exists[project][zone] = append(machineTypeCache.exists[project][zone], machineType)
	return true, nil
}
