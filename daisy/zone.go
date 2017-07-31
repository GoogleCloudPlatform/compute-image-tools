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
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var zones struct {
	valid []string
	mu    sync.Mutex
}

func checkZone(client compute.Client, project, zone string) error {
	zones.mu.Lock()
	defer zones.mu.Unlock()
	url := fmt.Sprintf("/project/%s/zone/%s", project, zone)
	if strIn(url, zones.valid) {
		return nil
	}
	if _, err := client.GetZone(project, zone); err != nil {
		return err
	}
	zones.valid = append(zones.valid, url)
	return nil
}
