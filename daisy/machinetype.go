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

var machineTypeURLRegex = regexp.MustCompile(fmt.Sprintf(`^(projects/(?P<project>%[1]s)/)?zones/(?P<zone>%[2]s)/machineTypes/(?P<machinetype>%[2]s)$`, projectRgxStr, rfc1035))

func (w *Workflow) machineTypeExists(project, zone, machineType string) (bool, DError) {
	predefinedMachineTypeExists, err := w.machineTypeCache.resourceExists(func(project, zone string, opts ...daisyCompute.ListCallOption) (interface{}, error) {
		return w.ComputeClient.ListMachineTypes(project, zone)
	}, project, zone, machineType)
	if err != nil {
		return false, err
	}
	if predefinedMachineTypeExists {
		return true, nil
	}

	// Check for custom machine types.
	w.machineTypeCache.mu.Lock()
	defer w.machineTypeCache.mu.Unlock()
	mt, cerr := w.ComputeClient.GetMachineType(project, zone, machineType)
	if cerr != nil {
		return false, typedErr(apiError, "failed to get machine type", cerr)
	}
	w.machineTypeCache.exists[project][zone][mt.Name] = mt
	return true, nil
}
