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
	"net/http"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
	"google.golang.org/api/googleapi"
)

var projectCache struct {
	exists []string
	mu     sync.Mutex
}

func projectExists(client compute.Client, project string) (bool, error) {
	projectCache.mu.Lock()
	defer projectCache.mu.Unlock()
	if strIn(project, projectCache.exists) {
		return true, nil
	}
	if _, err := client.GetProject(project); err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	projectCache.exists = append(projectCache.exists, project)
	return true, nil
}
