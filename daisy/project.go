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

var projects struct {
	valid []string
	mu    sync.Mutex
}

func checkProject(client compute.Client, project string) error {
	projects.mu.Lock()
	defer projects.mu.Unlock()
	if strIn(project, projects.valid) {
		return nil
	}
	if _, err := client.GetProject(project); err != nil {
		return err
	}
	projects.valid = append(projects.valid, project)
	return nil
}
