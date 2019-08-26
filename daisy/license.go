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

var licenseURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/licenses/(?P<license>%[2]s)$`, projectRgxStr, rfc1035))

var licenseCache struct {
	exists map[string][]string
	mu     sync.Mutex
}

func licenseExists(client compute.Client, project, license string) (bool, DError) {
	licenseCache.mu.Lock()
	defer licenseCache.mu.Unlock()
	if licenseCache.exists == nil {
		licenseCache.exists = map[string][]string{}
	}
	if _, ok := licenseCache.exists[project]; !ok || !strIn(license, licenseCache.exists[project]) {
		if _, err := client.GetLicense(project, license); err != nil {
			return false, typedErr(apiError, "failed to get license", err)
		}
		licenseCache.exists[project] = append(licenseCache.exists[project], license)
	}
	return true, nil
}
