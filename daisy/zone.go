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
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var zonesCache struct {
	exists map[string][]string
	mu     sync.Mutex
}

func zoneExists(client compute.Client, project, zone string) (bool, dErr) {
	zonesCache.mu.Lock()
	defer zonesCache.mu.Unlock()
	if zonesCache.exists == nil {
		zonesCache.exists = map[string][]string{}
	}
	if _, ok := zonesCache.exists[project]; !ok {
		zl, err := client.ListZones(project)
		if err != nil {
			return false, typedErr(apiError, err)
		}
		var zones []string
		for _, z := range zl {
			zones = append(zones, z.Name)
		}
		zonesCache.exists[project] = zones
	}
	return strIn(zone, zonesCache.exists[project]), nil
}
