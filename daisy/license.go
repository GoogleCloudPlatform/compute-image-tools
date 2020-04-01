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

	daisyCompute "github.com/GoogleCloudPlatform/compute-image-tools/daisy/compute"
)

var licenseURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?global/licenses/(?P<license>%[2]s)$`, projectRgxStr, rfc1035))

func (w *Workflow) licenseExists(project, license string) (bool, DError) {
	return w.licenseCache.resourceExists(func(project string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListLicenses(project)
	}, project, license)
}
