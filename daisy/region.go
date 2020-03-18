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
	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var regionsCache globalResourceCache

func regionExists(client compute.Client, project, region string) (bool, DError) {
	regionsCache.mu.Lock()
	defer regionsCache.mu.Unlock()
	if regionsCache.exists == nil {
		regionsCache.exists = map[string][]interface{}{}
	}
	if _, ok := regionsCache.exists[project]; !ok {
		rl, err := client.ListRegions(project)
		if err != nil {
			return false, typedErr(apiError, "failed to list regions", err)
		}
		var regions []interface{}
		for _, r := range rl {
			regions = append(regions, r.Name)
		}
		regionsCache.exists[project] = regions
	}
	return strInSlice(region, regionsCache.exists[project]), nil
}
